package listener

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type slashResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
	Username     string `json:"text"`
}

func init() {
	http.HandleFunc("/", handleMessage)
}

func handleMessage(w http.ResponseWriter, r *http.Request){

	if token != "" && r.PostFormValue("token") != token {
		http.Error(w, "Invalid Slack token.", http.StatusBadRequest)
		return
	}
	c := appengine.NewContext(r)
	log.Errorf(c, "Got token: %s", r.PostFormValue("token"))

	w.Header().Set("content-type", "application/json")

	resp := &slashResponse{
		ResponseType: "in_channel",
		Text:          "Kiitos "+r.PostFormValue("user_name")+ "! " + answers[rand.Intn(len(answers))],
		Username:      r.PostFormValue("text"),
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(c, "Error encoding JSON: %s", err)
		http.Error(w, "Error encoding JSON.", http.StatusInternalServerError)
		return
	}
}
