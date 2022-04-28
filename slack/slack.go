package slack

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"encore.dev/rlog"

	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack"
)

var (
	api *slack.Client
)


var secrets struct {
	AppToken      string // xapp
	BotToken      string // xoxb
	SigningSecret string // signing secret
}

func init() {
	rlog.Debug("Init method invoked")
	authenticateWithSlack()
}

func authenticateWithSlack() {
	rlog.Info("Initializing slack authentication")
	api = slack.New(secrets.BotToken,
		slack.OptionAppLevelToken(secrets.AppToken),
	)

	if _, err := api.AuthTest(); err != nil {
		rlog.Error("Got this err from AuthTest: ", "err", err)
		panic("issues: " + err.Error())
	}
	rlog.Info("Authenticated with Slack!")
}

//encore:api public raw path=/msgr
func Msgr(w http.ResponseWriter, r *http.Request) {
	body, eventsAPIEvent, idontKnowwhatThisVariableIs := parseEventFromBody(w, r)
	if idontKnowwhatThisVariableIs {
		return
	}

	parseEventsApiEvent(w, eventsAPIEvent, body)
}

func parseEventsApiEvent(w http.ResponseWriter, eventsAPIEvent slackevents.EventsAPIEvent, body []byte) {
	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent: 	consumeSlackCallBackEvent(w, body, eventsAPIEvent)
	case slackevents.URLVerification: 	slackURLVerification(w, body)
	default:
		rlog.Info("No case for event:", "eventsAPIEvent", eventsAPIEvent)
	}
}

func parseEventFromBody(w http.ResponseWriter, r *http.Request) ([]byte, slackevents.EventsAPIEvent, bool) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rlog.Error("failed opening body")
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if verifySlackSigning(w, r, body) {
		http.Error(w, "failed verifying slack signing token", http.StatusUnauthorized)
		return nil, slackevents.EventsAPIEvent{}, true
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		rlog.Error("Error parsing event :(")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, slackevents.EventsAPIEvent{}, true
	}
	return body, eventsAPIEvent, false
}

func consumeSlackCallBackEvent(w http.ResponseWriter, body []byte, event slackevents.EventsAPIEvent) {
	rlog.Debug("Event!", "event", event)
	innerEvent := event.InnerEvent

	switch ev := innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent: 	consumeAppMention(w, *ev)
	case *slackevents.MessageEvent: 	consumeMessageEvent(w, *ev, body)
	}
}

func consumeAppMention(w http.ResponseWriter, event slackevents.AppMentionEvent) {
	err := api.AddReaction("cake", slack.NewRefToMessage(event.Channel, event.TimeStamp))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}


}

func consumeMessageEvent(w http.ResponseWriter, event slackevents.MessageEvent, body []byte) {
	err := api.AddReaction("mrclean", slack.NewRefToMessage(event.Channel, event.TimeStamp))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}


	fileName := strings.Replace(event.Text," ", "_", -1)

	f, err := os.Create(fileName + ".json")
	if err != nil {
		rlogAndHttpError("Error creating the file", w, err, http.StatusInternalServerError)
	}
	if _, err := f.Write(body); err != nil {
		rlogAndHttpError("Error writing to the file", w, err, http.StatusInternalServerError)
	}

}

func slackURLVerification(w http.ResponseWriter, body []byte) {
	var r *slackevents.ChallengeResponse
	err := json.Unmarshal(body, &r)
	if err != nil {
		rlogAndHttpError("", w, err, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text")
	_, err = w.Write([]byte(r.Challenge))
	if err != nil {
		rlogAndHttpError("Error writing challenge response: ", w, err, http.StatusInternalServerError)
	}
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

func rlogAndHttpError(message string, w http.ResponseWriter, err error, statusCode int) {
	rlog.Error(message, "err", err)
	http.Error(w, message + err.Error(), statusCode)
}