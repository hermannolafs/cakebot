package slack

import (
	"encoding/json"
	"github.com/slack-go/slack/slackevents"
	"io/ioutil"
	"net/http"

	"github.com/slack-go/slack"
)

var api = slack.New(secrets.AppToken,
	slack.OptionAppLevelToken(secrets.BotToken),
	slack.OptionDebug(true),
)

var secrets struct {
	AppToken      string // xapp
	BotToken      string // xoxb
	SigningSecret string // signing secretttt
}

//encore:api public raw path=/cakebot
func CoolThing(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Verify secret here
	sv, err := slack.NewSecretsVerifier(r.Header, secrets.SigningSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// I am not sure
	if _, err := sv.Write(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Getting the event, not sure why noVerifyToken thing
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
}
