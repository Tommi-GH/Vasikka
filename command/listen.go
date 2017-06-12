package listener

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"strings"
	"io"
	"fmt"
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

	w.Header().Set("content-type", "application/json")

	sender := r.PostFormValue("user_name")
	message := strings.Replace(r.PostFormValue("text"),`"`,"´´",-1)

	att := &attachments{
		Text: message,
	}

	var attJson = att

	resp := &slashResponse{
		ResponseType: "ephemeral",
		Text:         "Kiitos " + sender + "! " + answers[rand.Intn(len(answers))],
		Attachments:  []*attachments{attJson, },
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf(c, "Error encoding JSON: %s", err)
		http.Error(w, "Error encoding JSON.", http.StatusInternalServerError)
		return
	}

	print(json.NewEncoder(w).Encode(resp))


	payload := strings.NewReader("{\"text\":\""+message+"\"}")
	sendRequest(r,slackurl,"application/json",payload)

	payload2 := strings.NewReader("entry.2059036820=Kokkavartio&entry.1364708498=Hyvin%20menee%20joo&entry.1911721708=Tommi%20T")
	sendRequest(r,formurl,"application/x-www-form-urlencoded",payload2)

}

func sendRequest(r *http.Request, url string, contentType string, payload io.Reader){

	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Set("content-type", contentType)
	log.Debugf(ctx,"%s",formatRequest(req))
	resp2, err2 := client.Do(req)

	log.Debugf(ctx,"%s",resp2)
	log.Errorf(ctx,"%s",err2)
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