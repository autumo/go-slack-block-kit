package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	config := getSecret()
	client := slack.New(
		config.OauthToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)
	log.SetOutput(os.Stdout)
	listener := SlackListener{client: client, botID: os.Getenv("BOT_ID")}
	go listener.ListenAndResponse()

	http.Handle("/interaction", interactionHandler{
		verificationToken:   config.VerificationToken,
		jenkinsBotUserToken: config.JenkinsBotUserToken,
		jenkinsJobToken:     config.JenkinsJobToken,
		githubBotUserToken:  config.GitHubBotUserToken,
		client:              client,
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})
	http.ListenAndServe(":3000", nil)
}
