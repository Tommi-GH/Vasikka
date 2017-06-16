package listener

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"

	"fmt"
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

func init() {
	http.HandleFunc("/", handleMessage)
}

func handleMessage(w http.ResponseWriter, r *http.Request) {

	var token = tokentest

	if !appengine.IsDevAppServer() {
		token = tokenprod
	}

	if token != "" && r.PostFormValue("token") != token {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		return
	}

	c := appengine.NewContext(r)

	w.Header().Set("content-type", "application/json")

	sender := r.PostFormValue("user_name")
	message := strings.Replace(r.PostFormValue("text"), `"`, "´´", -1)

	att := &attachments{
		Text: message,
	}

	var attJSON = att
	var resp = &slashResponse{}

	if saveDataToSheets(r, sender, message) == "" {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         "Kiitos " + sender + "! " + answers[rand.Intn(len(answers))],
			Attachments:  []*attachments{attJSON},
		}
	} else if saveDataToSheets(r, sender, message) == "noTarget" {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         noTargetMessage,
			Attachments:  []*attachments{attJSON},
		}
	} else {
		resp = &slashResponse{
			ResponseType: "ephemeral",
			Text:         errorMessage,
			Attachments:  []*attachments{attJSON},
		}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(c, "Error encoding JSON: %s", err)
		http.Error(w, "Error encoding JSON.", http.StatusInternalServerError)
		return
	}

	print(json.NewEncoder(w).Encode(resp))

	payload := strings.NewReader("{\"text\":\"" + message + "\"}")
	sendRequest(r, slackurl, "application/json", payload)

}

func sendRequest(r *http.Request, targeturl string, contentType string, payload io.Reader) {

	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	req, _ := http.NewRequest("POST", targeturl, payload)
	req.Header.Set("Content-Type", contentType)
	log.Debugf(ctx, "%s", formatRequest(req))
	resp2, err2 := client.Do(req)

	log.Debugf(ctx, "Vastaus: %s", resp2)
	muuttuja, _ := ioutil.ReadAll(resp2.Body)
	log.Errorf(ctx, string(muuttuja))
	log.Errorf(ctx, "Virheviesti: %s", err2)
	defer resp2.Body.Close()

}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string

	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)

	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))

	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}

	// Return the request as a string
	return strings.Join(request, "\n")
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

	valueInputOption := "RAW"
	writeRange := "Sheet1!A1"
	var vr sheets.ValueRange

	targets, err := srv.Spreadsheets.Values.Get(targetSpreadsheetID, "Sheet1!A2:B").Context(ctx).Do()

	if err != nil {
		log.Errorf(ctx, "Unable to retrieve data from targetsheet. %v", err)
		return "Error"
	}

	var target = parseTarget(message, targets)
	if target == "" {
		return "noTarget"
	}

	myval := []interface{}{time.Now(), target, message, sender}
	vr.Values = append(vr.Values, myval)

	_, err = srv.Spreadsheets.Values.Append(reportSpreadsheetID, writeRange, &vr).ValueInputOption(valueInputOption).Context(ctx).Do()
	if err != nil {
		log.Errorf(ctx, "Unable to retrieve data from reportsheet. %v", err)
		return "Error"
	}

	return ""

}

func parseTarget(message string, lippukunnat *sheets.ValueRange) string {

	if len(lippukunnat.Values) > 0 {

		fullname := ""
		shortname := ""
		hasShortName := false
		hasLongName := false

		for _, row := range lippukunnat.Values {

			fullname = row[0].(string)
			if len(row) > 1 {
				shortname = row[1].(string)
			}

			if len(fullname) > 1 {
				hasLongName = strings.Contains(strings.ToLower(message), strings.ToLower(fullname))
			}

			if len(shortname) > 1 {
				hasShortName = strings.Contains(strings.ToLower(message), strings.ToLower(shortname+" "))
			}

			if len(shortname) > 1 {
				hasShortName = strings.Contains(strings.ToLower(message), strings.ToLower(" "+shortname))
			}

			if hasShortName || hasLongName {
				return fullname
			}
			hasShortName = false
			hasLongName = false
		}
	}

	return ""
}
