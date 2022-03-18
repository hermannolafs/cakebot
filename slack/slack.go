package slack

import (
	"encoding/json"
	"encore.dev/rlog"
	"github.com/slack-go/slack/slackevents"
	"io/ioutil"
	"net/http"

	"github.com/slack-go/slack"
)

const cowart = "Moo! %s"

var api *slack.Client

var secrets struct {
	AppToken      string // xapp
	BotToken      string // xoxb
	SigningSecret string // signing secretttt
}

func init() {
	rlog.Debug("Init method invoked")
	api = slack.New(secrets.AppToken,
		slack.OptionAppLevelToken(secrets.BotToken),
		slack.OptionDebug(true),
	)

	test, err := api.AuthTest()
	rlog.Debug("Got this output from AuthTest: ", "test", test)
	rlog.Debug("Got this err from AuthTest: ", "err", err)
	if err != nil {
		panic("issues: " + err.Error())
	}

}

//encore:api public raw path=/msgr
func Msgr(w http.ResponseWriter, r *http.Request)  {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if verifySlackSigning(w, r, body) {
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent:
		consumeSlackCallBackEvent(w, eventsAPIEvent)
	case slackevents.URLVerification:
		slackURLVerification(w, body)
	default:
		rlog.Debug("No case for event:", "eventsAPIEvent", eventsAPIEvent)
	}
}

func consumeSlackCallBackEvent(w http.ResponseWriter, event slackevents.EventsAPIEvent) {
	innerEvent := event.InnerEvent
	switch ev := innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		_, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText("Yes", false))
		if err != nil {
			rlog.Debug(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func slackURLVerification(w http.ResponseWriter, body []byte) {
	rlog.Debug("Got a url verify request", )
	var r *slackevents.ChallengeResponse
	err := json.Unmarshal([]byte(body), &r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text")
	w.Write([]byte(r.Challenge))
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