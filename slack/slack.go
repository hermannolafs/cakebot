package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"encore.dev/rlog"

	"github.com/slack-go/slack"
	"github.com/buger/jsonparser"
)

const cowart = "Moo! %s"

var api = slack.New(secrets.AppToken,
	slack.OptionAppLevelToken(secrets.BotToken),
	slack.OptionDebug(true),
)

var secrets struct {
	AppToken      string // xapp
	BotToken      string // xoxb
	SigningSecret string // signing secretttt
}

//encore:api public raw path=/simpler
func Simpler(w http.ResponseWriter, r *http.Request) {
	body, fucked := readBodyFromRequest(w, r)
	if fucked {
		return
	}

	rlog.Debug("Got this body:" + string(body))

	if verifySlackSigning(w, r, body) {
		return
	}

	text, err := jsonparser.GetString(body, "text")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	data, _ := json.Marshal(map[string]string{
		"response_type": "in_channel",
		"text":          fmt.Sprintf("Scibbiddy BOOO: %s", text),
	})
	rlog.Debug("Got this text:" + text)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func readBodyFromRequest(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, true
	}
	return body, false
}

// Example fetched from here:
// https://github.com/slack-go/slack/blob/master/examples/eventsapi/events.go
// Returning bool because intelliJ autoextract method said so
// TODO refactor this to be more clever
func verifySlackSigning(w http.ResponseWriter, r *http.Request, body []byte) bool {
	secretsVerifier, err := slack.NewSecretsVerifier(r.Header, secrets.SigningSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return true
	}
	if _, err := secretsVerifier.Write(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	if err := secretsVerifier.Ensure(); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return true
	}
	return false
}

//
////encore:api public raw path=/cakebot
//func CoolThing(w http.ResponseWriter, r *http.Request) {
//
//	body, err := ioutil.ReadAll(r.Body)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//	// Verify secret here
//	sv, err := slack.NewSecretsVerifier(r.Header, secrets.SigningSecret)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//	// I am not sure
//	if _, err := sv.Write(body); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	if err := sv.Ensure(); err != nil {
//		http.Error(w, err.Error(), http.StatusUnauthorized)
//		return
//	}
//
//	// Getting the event, not sure why noVerifyToken thing
//	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	//
//	if eventsAPIEvent.Type == slackevents.URLVerification {
//		var r *slackevents.ChallengeResponse
//		err := json.Unmarshal([]byte(body), &r)
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//		w.Header().Set("Content-Type", "text")
//		w.Write([]byte(r.Challenge))
//	}
//	if eventsAPIEvent.Type == slackevents.CallbackEvent {
//		innerEvent := eventsAPIEvent.InnerEvent
//		switch ev := innerEvent.Data.(type) {
//		case *slackevents.AppMentionEvent:
//			api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
//		}
//	}
//
//	api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false), slack.M)
//
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(200)
//}
