package listener

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"strings"
	"net/url"
)

type attachments struct {
	Text         string `json:"text"`
}

type slashResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
	Attachments  []*attachments `json:"attachments"`
}

func init() {
	http.HandleFunc("/", handleMessage)
}

func handleMessage(w http.ResponseWriter, r *http.Request) {

	var token= tokentest

	if !appengine.IsDevAppServer() {
		token = tokenprod
	}

	if token != "" && r.PostFormValue("token") != token {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		return
	}

	c := appengine.NewContext(r)
	log.Errorf(c, "Got token: %s", r.PostFormValue("token"))

	w.Header().Set("content-type", "application/json")

	att := &attachments{
		Text: r.PostFormValue("text"),
	}

	var attJson = att

	resp := &slashResponse{
		ResponseType: "in_channel",
		Text:         "Kiitos " + r.PostFormValue("user_name") + "! " + answers[rand.Intn(len(answers))],
		Attachments:  []*attachments{attJson, },
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(c, "Error encoding JSON: %s", err)
		http.Error(w, "Error encoding JSON.", http.StatusInternalServerError)
		return
	}

	print(json.NewEncoder(w).Encode(resp))

	//sendForm

}

//How is sending a POST-request from AppEngine supposed to work...
func sendForm() {

	payload := url.Values{qtarget: {"asdf"}, qtext: {"asdf asdf"}, qreporter: {"asdf sadf"}}

	r,_ := http.NewRequest("POST",formurl,strings.NewReader(payload.Encode()))


	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	client.Do(r)

}