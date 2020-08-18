package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"

	"strings"

	sheets "google.golang.org/api/sheets/v4"
)

type attachments struct {
	Text string `json:"text"`
}

type slashResponse struct {
	ResponseType string         `json:"response_type"`
	Text         string         `json:"text"`
	Attachments  []*attachments `json:"attachments"`
}

//Info for team is given in a separate .go file and saved to
//a map in configuration .go file
type teamInfo struct {
	PrivateSlackurl     string
	PublicSlackurl      string `text:"omitempty"`
	TargetSpreadsheetID string
	ReportSpreadsheetID string
	ErrorMessage        string
	NoTargetMessage     string
	Answer              string
	ReadRange           string
	WriteRange          string
	NoPostKey           string
	HelpText            string
	AnswerPrefix        string `text:"omitempty"`
}

var team teamInfo
var indexTmpl = template.Must(template.ParseFiles("index.html"))

func main() {
	http.HandleFunc("/", handleMessage)
	http.HandleFunc("/index.html", handleIndex)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	//log.Infof("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, nil); err != nil {
		c := r.Context()
		log.Fatal(c, "Error executing indexTmpl template: %s", err)
	}
}

//the main function for POST-request handling
func handleMessage(w http.ResponseWriter, r *http.Request) {

	log.Printf("A new request!")
	token := r.PostFormValue("token")
	team = getTeamInfo(token)

	//if team info is not found, the token is invalid
	//team info is in a map-object in the configuration file
	//with token as key
	if len(team.PublicSlackurl) == 0 {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		log.Printf("Invalid Slack token")
		return
	}

	ctx := r.Context()
	w.Header().Set("content-type", "application/json")

	//escape problematic characters
	message := strings.Replace(strings.Replace(r.PostFormValue("text"), `"`, "´´", -1), "\\", "/", -1)

	//If the request is a valid report, do the following steps,
	//else return appropriate error-message
	resp, isValid := createResponse(r, message)
	if isValid {
		sendSlackMsg(message, r, true)
		saveDataToSheets(r, message)
	}

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Fatal(ctx, "Error encoding JSON: %s", err)
	}
}

//Creates the response for the initial POST-request. The response
//includes an ephemeral slack-message
func createResponse(r *http.Request, message string) (*slashResponse, bool) {

	//If the message is -help, return the helptext gicen in the team info
	if strings.EqualFold(strings.ToLower(message), "-help") {
		return &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.HelpText,
		}, false
	}

	//If the message starts with -report, return a list of reports for given
	//target or all targets if target is "all"
	if strings.HasPrefix(strings.ToLower(message), "-report") {
		return &slashResponse{
			ResponseType: "ephemeral",
			Text:         getTargetReports(r, message),
			Attachments: []*attachments{&attachments{
				Text: "Report query: " + message,
			}},
		}, false
	}

	//Find the target in message
	target := findTarget(r, message)

	//If target is not found, respond with team specific message
	if target == "noTarget" {
		return &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.NoTargetMessage,
			Attachments: []*attachments{&attachments{
				Text: message,
			}},
		}, false
	}

	//Target found in message, the request is a valid report
	if len(target) > 0 {

		return &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.AnswerPrefix + r.PostFormValue("user_name") + "! " + team.Answer,
			Attachments: []*attachments{&attachments{
				Text: message,
			}},
		}, true

	}

	return &slashResponse{
		ResponseType: "ephemeral",
		Text:         team.ErrorMessage,
		Attachments: []*attachments{&attachments{
			Text: message,
		}},
	}, false
}

// Send Slack message to dedicated channel as bot user
func sendSlackMsg(message string, r *http.Request, public bool) {

	if strings.Contains(message, team.NoPostKey) {
		return
	}

	payload := strings.NewReader("{\"text\":\"" + message + "\"}")

	ctx := r.Context()
	client := &http.Client{}

	//for separate channels depending on keywords in message, not in use currently
	/*if public {
		req, _ := http.NewRequest("POST", team.PublicSlackurl, payload)
	} else {
		req, _ := http.NewRequest("POST", team.PrivateSlackurl, payload)
	}*/

	req, _ := http.NewRequest("POST", team.PublicSlackurl, payload)
	req.Header.Set("Content-Type", "application/json")

	_, err := client.Do(req)

	if err != nil {
		log.Fatal(ctx, "Unable to send message as bot user: %s", err)
	}

}

//Writes timestamp, target name, message and sender name to team-specific Google sheet
func saveDataToSheets(r *http.Request, message string) string {

	ctx := r.Context()
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatal(ctx, "Unable to create client %s", err)
		return "Error"
	}

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve Sheets Client %v", err)
		return "Error"
	}

	target := findTarget(r, message)

	layout := "01/02/2006 15:04:05"
	timestamp := time.Now().Format(layout)

	valueInputOption := "RAW"
	var vr sheets.ValueRange

	saveData := []interface{}{timestamp, target, message, r.PostFormValue("user_name")}
	vr.Values = append(vr.Values, saveData)

	_, err = srv.Spreadsheets.Values.Append(team.ReportSpreadsheetID, team.WriteRange, &vr).ValueInputOption(valueInputOption).Context(ctx).Do()
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve data from reportsheet. %v", err)
		return "Error"
	}

	return ""

}

//Looks for targets in message. Targets are listed in team-specific Google sheet
//returns the target's name or "noTarget" if target wasn't found and empty string
//if there's an error reading the target list
func findTarget(r *http.Request, message string) string {

	ctx := r.Context()
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatal(ctx, "Unable to create client %s", err)
		return ""
	}

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve Sheets Client %v", err)
		return ""
	}

	targets, err := srv.Spreadsheets.Values.Get(team.TargetSpreadsheetID, team.ReadRange).Context(ctx).Do()
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve data from targetsheet. %v", err)
		return ""
	}

	if len(targets.Values) > 0 {

		fullname := ""
		shortname := ""

		for _, row := range targets.Values {

			fullname = row[0].(string)
			if len(row) > 1 {
				shortname = row[1].(string)
			}

			if len(fullname) > 1 &&
				strings.Contains(strings.ToLower(message), strings.ToLower(fullname)) {
				return fullname
			}

			if len(shortname) > 1 {
				if strings.HasPrefix(strings.ToLower(message), strings.ToLower(shortname+" ")) {
					return fullname
				}
				if strings.HasSuffix(strings.ToLower(message), strings.ToLower(" "+shortname)) {
					return fullname
				}
				if strings.Contains(strings.ToLower(message), strings.ToLower(" "+shortname+" ")) {
					return fullname
				}
			}
		}
	}

	return "noTarget"

}

//finds reports on the given target or all reports if target is "all"
//return all reports formatted into one string
func getTargetReports(r *http.Request, message string) string {

	ctx := r.Context()
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatal(ctx, "Unable to create client %s", err)
		return "Error"
	}

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve Sheets Client %v", err)
		return "Error"
	}

	data, err := srv.Spreadsheets.Values.Get(team.ReportSpreadsheetID, team.WriteRange).Context(ctx).Do()
	if err != nil {
		log.Fatal(ctx, "Unable to retrieve data from targetsheet. %v", err)
		return "Error"
	}

	target := ""

	splitMessage := strings.Split(message, " ")
	if len(splitMessage) > 1 {
		target = splitMessage[1]
	}

	if strings.EqualFold(target, "all") {
		target = "all"
	} else {
		target = findTarget(r, message)
	}

	if strings.EqualFold(target, "noTarget") {
		return team.NoTargetMessage
	}

	var reports = []string{}

	if len(data.Values) > 0 {

		for _, row := range data.Values {

			if strings.EqualFold(target, "all") || strings.EqualFold(strings.ToLower(row[1].(string)), target) {
				reports = append(reports, row[2].(string)+" Reporter: "+row[3].(string))
			}
		}
	}

	if len(reports) == 0 {
		return "No reports found"
	}

	targetReports := "Reports for target " + target + "\n"

	for _, report := range reports {

		targetReports = targetReports + report + "\n"

	}

	return targetReports
}
