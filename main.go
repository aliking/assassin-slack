package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	tokenConfig   = "INCOMING_SLACK_TOKEN"
	webhookConfig = "INCOMING_SLACK_WEBHOOK"
	// Incoming payload form will have the following keys:
	// (See: https://api.slack.com/slash-commands)
	keyToken       = "token"
	keyTeamID      = "team_id"
	keyChannelID   = "channel_id"
	keyChannelName = "channel_name"
	keyUserID      = "user_id"
	keyUserName    = "user_name"
	keyCommand     = "command"
	keyText        = "text"
	slackchannel   = "#assassins"
)

type slackMsg struct {
	Text      string `json:"text"`
	Username  string `json:"username"`
	Channel   string `json:"channel"` // Recipient
	AsUser    string `json:"as_user"`
	IconURL   string `json:"icon_url"`
	LinkNames string `json:"link_names"`
}

var (
	port      int
	assassins = make(map[string][]string)
	icon      string
	name      string
)

// readAnonymousMessage parses the username and re-routes
// the message to the user from an anonymous animal
func readAnonymousMessage(r *http.Request) string {
	err := r.ParseForm()
	// TODO: Change HTTP status code
	if err != nil {
		return string(err.Error())
	}
	// Incoming POST's token should match the one set in Heroku
	if len(r.Form[keyToken]) == 0 || r.Form[keyToken][0] != os.Getenv(tokenConfig) {
		return "Config error."
	}
	if len(r.Form[keyText]) == 0 {
		return "Slack bug; inform the team."
	}
	msg := strings.TrimSpace(r.Form[keyText][0])

	user := r.Form[keyUserName][0]
	err = sendAnonymousMessage(user, msg)
	if err != nil {
		return "Failed to send message."
	}
	return fmt.Sprintf("Anonymously sent [%s] to %s", msg, user)
}

// sendAnonymousMessage uses an incoming hook to Direct Message
// the given user the message, from the registered assassin
func sendAnonymousMessage(username, message string) error {
	url := os.Getenv(webhookConfig)

	if len(assassins[username]) != 0 {
		icon = assassins[username][1]
		name = assassins[username][0]
	} else {
		icon = "http://i.imgur.com/CyIgnqi.png"
		name = "Civilian"
	}

	fmt.Println("%s %s", icon, name)
	payload, err := json.Marshal(slackMsg{
		Text:      message,
		Channel:   slackchannel,
		AsUser:    "False",
		IconURL:   icon,
		LinkNames: "1",
		Username:  name,
	})
	if err != nil {
		return err
	}

	_, err = http.Post(url, "application/json", bytes.NewBuffer(payload))
	return err
}

func main() {
	getAssassins()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		result := readAnonymousMessage(r)
		fmt.Fprintf(w, result)
	})
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func init() {
	flag.IntVar(&port, "port", 5000, "HTTP server port")
	flag.Parse()
}

// Assassin data structure
type Assassin struct {
	Username     string `json:"username"`
	AssassinName string `json:"assassin_name"`
	IconURL      string `json:"icon_url"`
}

func getAssassins() []Assassin {
	raw, err := ioutil.ReadFile("./assassins.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var c []Assassin
	json.Unmarshal(raw, &c)
	for _, p := range c {
		fmt.Println(p.Username)
		assassins[p.Username] = []string{p.AssassinName, p.IconURL}
	}
	return c
}
