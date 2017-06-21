package listener

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"

	"strings"

	sheets "google.golang.org/api/sheets/v4"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

type attachments struct {
	Text string `json:"text"`
}

type slashResponse struct {
	ResponseType string         `json:"response_type"`
	Text         string         `json:"text"`
	Attachments  []*attachments `json:"attachments"`
}

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
}

var team teamInfo

func init() {
	http.HandleFunc("/", handleMessage)
}

func handleMessage(w http.ResponseWriter, r *http.Request) {

	team = getTeamInfo(r.PostFormValue("token"))

	if len(team.PublicSlackurl) == 0 {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		return
	}

	c := appengine.NewContext(r)

	w.Header().Set("content-type", "application/json")

	sender := r.PostFormValue("user_name")
	message := strings.Replace(strings.Replace(r.PostFormValue("text"), `"`, "´´", -1), "\\", "/", -1)

	att := &attachments{
		Text: message,
	}

	var attJSON = att
	var resp = &slashResponse{}

	saveMessageResp := saveDataToSheets(r, sender, message)

	if strings.EqualFold(strings.ToLower(message), "-help") {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.HelpText,
		}
	} else if strings.HasPrefix(strings.ToLower(message), "-hae") {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         getTargetReports(r, message),
		}
	} else if saveMessageResp == "" {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         "Kiitos " + sender + "! " + team.Answer,
			Attachments:  []*attachments{attJSON},
		}

		sendSlackMsg(message, r, true)

	} else if saveMessageResp == "noTarget" {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.NoTargetMessage,
			Attachments:  []*attachments{attJSON},
		}
	} else {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         team.ErrorMessage,
			Attachments:  []*attachments{attJSON},
		}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(c, "Error encoding JSON: %s", err)
	}

}

// Send Slack message to dedicated channel as bot user
func sendSlackMsg(message string, r *http.Request, public bool) {

	if !strings.Contains(message, team.NoPostKey) {
		return
	}

	payload := strings.NewReader("{\"text\":\"" + message + "\"}")

	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	/*if public {
		req, _ := http.NewRequest("POST", team.PublicSlackurl, payload)
	} else {
		req, _ := http.NewRequest("POST", team.PrivateSlackurl, payload)
	}*/

	req, _ := http.NewRequest("POST", team.PublicSlackurl, payload)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req)

	if err != nil {
		log.Errorf(ctx, "Unable to send message as bot user: %s", err)
	}

	defer resp2.Body.Close()

}

func saveDataToSheets(r *http.Request, sender string, message string) string {

	ctx := appengine.NewContext(r)
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Errorf(ctx, "Unable to create client %s", err)
		return "Error"
	}

	srv, err := sheets.New(client)
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve Sheets Client %v", err)
		return "Error"
	}

	targets, err := srv.Spreadsheets.Values.Get(team.TargetSpreadsheetID, team.ReadRange).Context(ctx).Do()
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve data from targetsheet. %v", err)
		return "Error"
	}

	var target = parseKeywords(message, targets)
	if target == "" {
		return "noTarget"
	}

	layout := "01/02/2006 15:04:05"
	timestamp := time.Now().Format(layout)

	valueInputOption := "RAW"
	var vr sheets.ValueRange

	myval := []interface{}{timestamp, target, message, sender}
	vr.Values = append(vr.Values, myval)

	_, err = srv.Spreadsheets.Values.Append(team.ReportSpreadsheetID, team.WriteRange, &vr).ValueInputOption(valueInputOption).Context(ctx).Do()
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve data from reportsheet. %v", err)
		return "Error"
	}

	return ""

}

func parseKeywords(message string, keywords *sheets.ValueRange) string {

	if len(keywords.Values) > 0 {

		fullname := ""
		shortname := ""

		for _, row := range keywords.Values {

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

	return ""
}

func getTargetReports(r *http.Request, message string) string {

	ctx := appengine.NewContext(r)
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Errorf(ctx, "Unable to create client %s", err)
		return "Error"
	}

	srv, err := sheets.New(client)
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve Sheets Client %v", err)
		return "Error"
	}

	data, err := srv.Spreadsheets.Values.Get(team.ReportSpreadsheetID, "B2:D").Context(ctx).Do()
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve data from targetsheet. %v", err)
		return "Error"
	}

	target := strings.ToLower(message[strings.Index(message, " "):len(message)])
	reports := []string{""}

	if len(data.Values) > 0 {

		for _, row := range data.Values {

			if strings.EqualFold(strings.ToLower(row[0].(string)), target) {
			}
			reports = append(reports, row[1].(string)+" Reporter: "+row[2].(string))
		}
	}

	if len(reports) == 0 {
		return "No reports found"
	}

	targetReports := "Reports for target" + target + "\n"

	for _, report := range reports {

		targetReports = targetReports + report + "\n"

	}

	return targetReports
}
