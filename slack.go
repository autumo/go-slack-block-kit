package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

type SlackListener struct {
	client *slack.Client
	botID  string
}

func (s *SlackListener) ListenAndResponse() {
	// Start listening slack events
	rtm := s.client.NewRTM()
	go rtm.ManageConnection()

	// Handle slack events
	for msg := range rtm.IncomingEvents {
		fmt.Println("Event Reveived: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				log.Printf("[ERROR] Failed to handle message: %s", err)
			}
			log.Print("[INFO] call")
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent) error {
	// Only response mention to bot. Ignore else.
	log.Print(ev.Msg.Text)
	if !strings.HasPrefix(ev.Msg.Text, fmt.Sprintf("<@%s> ", s.botID)) {
		return nil
	}
	if regexp.MustCompile(`deploy staging`).MatchString(ev.Msg.Text) {
		msgOpt := SelectDeployTarget("staging")
		s.client.PostMessage(ev.Msg.Channel, msgOpt)
		return nil
	}
	if regexp.MustCompile(`deploy production`).MatchString(ev.Msg.Text) {
		msgOpt := SelectDeployTarget("production")
		s.client.PostMessage(ev.Msg.Channel, msgOpt)
		return nil
	}
	return nil
}

func SelectDeployTarget(phase string) slack.MsgOption {
	headerText := slack.NewTextBlockObject("mrkdwn", ":jenkins:", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)

	apiSection := createDeployButtonSection("API", API, phase)
	authSection := createDeployButtonSection("Auth", Auth, phase)

	closeAction := CloseButtonAction()

	return slack.MsgOptionBlocks(
		headerSection,
		apiSection,
		authSection,
		closeAction,
	)
}

func createDeployButtonSection(summary string, target Project, phase string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", "*"+summary+"*", false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("deploy_select_%s_%s", target, phase), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return section
}

func CloseButtonAction() *slack.ActionBlock {
	closeBtnTxt := slack.NewTextBlockObject("plain_text", "Close", false, false)
	closeBtn := slack.NewButtonBlockElement("", "close", closeBtnTxt)
	section := slack.NewActionBlock("", closeBtn)
	return section
}

type Project string

const (
	API  Project = "API"
	Auth Project = "Auth"
)

func (p Project) JenkinsJob() string {
	return fmt.Sprintf("Deploy-%s", p)
}

func (p Project) GitHubRepository() string {
	return fmt.Sprintf("go-%s", strings.ToLower(string(p)))
}
