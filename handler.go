package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

type interactionHandler struct {
	verificationToken   string
	jenkinsBotUserToken string
	jenkinsJobToken     string
	githubBotUserToken  string
	client              *slack.Client
}

func getSlackError(system, msg string, user string) []byte {
	respoonse := slack.Message{
		Msg: slack.Msg{
			ResponseType: "in_channel",
			Text:         fmt.Sprintf("%s: %s actioned by <@%s>", system, msg, user),
		},
	}
	respoonse.ReplaceOriginal = true
	responseBytes, _ := json.Marshal(respoonse)

	return responseBytes
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse input from request
	if err := r.ParseForm(); err != nil {
		// getSlackError is a helper to quickly render errors back to slack
		responseBytes := getSlackError("Server Error", "An unknown error occurred", "unkown")
		w.Write(responseBytes) // not display message on slack
		return
	}

	interactionRequest := slack.InteractionCallback{}
	json.Unmarshal([]byte(r.PostForm.Get("payload")), &interactionRequest)

	// Get the action from the request, it'll always be the first one provided in my case
	var actionValue string
	switch interactionRequest.ActionCallback.BlockActions[0].Type {
	case "button":
		actionValue = interactionRequest.ActionCallback.BlockActions[0].Value
	case "static_select":
		actionValue = interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Value
	}
	userID := interactionRequest.User.ID
	// Handle close action
	if strings.Contains(actionValue, "close") {

		// Found this on stack overflow, unsure if this exists in the package
		closeStr := fmt.Sprintf(`{
		'response_type': 'in_channel',
		'text': 'closed by <@%s>',
		'replace_original': true,
		'delete_original': true
		}`, userID)

		// Post close json back to response URL to close the message
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer([]byte(closeStr)))
		return
	}
	log.Printf("[INFO] Action Value: %s", actionValue)
	switch {
	case strings.HasPrefix(actionValue, "deploy_select_"):
		h.DeployInteraction(w, interactionRequest)
	case strings.HasPrefix(actionValue, "select_branch_value_"):
		h.DeployCheckInteraction(w, interactionRequest)
	case strings.HasPrefix(actionValue, "ok_select_branch_value_"):
		h.SelectBranchInteraction(w, interactionRequest)
	default:
		log.Print("[ERROR] An unknown error occurred")
		responseBytes := getSlackError("Server Error", "An unknown error occurred", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer([]byte(responseBytes)))
	}
}

func (h interactionHandler) deployResponse(res string, userID string) slack.Message {
	blockObject := slack.NewTextBlockObject("mrkdwn", res, false, false)
	sectionBlock := slack.NewSectionBlock(blockObject, nil, nil)
	userObject := slack.NewTextBlockObject("mrkdwn", "by <@"+userID+">", false, false)
	userBlock := slack.NewSectionBlock(userObject, nil, nil)

	msg := slack.NewBlockMessage(
		sectionBlock,
		userBlock,
	)

	return msg
}

func (h interactionHandler) DeployInteraction(w http.ResponseWriter, interactionRequest slack.InteractionCallback) {
	actionValue := interactionRequest.ActionCallback.BlockActions[0].Value
	userID := interactionRequest.User.ID
	if !IsDeveploers(userID) {
		log.Print("[ERROR] Forbidden Error")
		responseBytes := getSlackError("Forbidden Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	arr := strings.Split(actionValue, "_")
	if len(arr) != 4 {
		log.Print("[ERROR] Internal Server Error")
		responseBytes := getSlackError("Internal Server Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	responseData := h.selectBranchList(Project(arr[2]), arr[3])
	responseData.ReplaceOriginal = true
	responseBytes, _ := json.Marshal(responseData)

	http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))

	return
}

func (h interactionHandler) selectBranchList(z Project, phase string) slack.Message {
	repo := z.GitHubRepository()
	github := CreateGitHubInstance(h.githubBotUserToken)
	arr, err := github.ListBranch(repo)
	if err != nil {
		log.Print("Failed to list branch" + err.Error())
	}
	var opts []*slack.OptionBlockObject
	for i, v := range arr {
		txt := slack.NewTextBlockObject("plain_text", v, false, false)
		opt := slack.NewOptionBlockObject(fmt.Sprintf("select_branch_value_%s_%s_%d", z, phase, i), txt)
		opts = append(opts, opt)
	}
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* branch list", repo), false, false)
	availableOption := slack.NewOptionsSelectBlockElement("static_select", nil, "", opts...)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(availableOption))
	closeAction := CloseButtonAction()

	return slack.NewBlockMessage(
		section,
		closeAction,
	)
}

func (h interactionHandler) DeployCheckInteraction(w http.ResponseWriter, interactionRequest slack.InteractionCallback) {
	actionValue := interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Value
	userID := interactionRequest.User.ID
	if !IsDeveploers(userID) {
		log.Print("[ERROR] Forbidden Error")
		responseBytes := getSlackError("Forbidden Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	arr := strings.Split(actionValue, "_")
	if len(arr) != 6 { // select_branch_value_%s_%s_%d
		log.Print("[ERROR] Internal Server Error")
		responseBytes := getSlackError("Internal Server Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	responseData := h.deployButton(arr[3], arr[4], interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Text.Text)
	responseData.ReplaceOriginal = true
	responseBytes, _ := json.Marshal(responseData)

	http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))

	return
}

func (h interactionHandler) deployButton(target string, phase string, branch string) slack.Message {
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\n*%s* ブランチ\nをデプロイしますか?", Project(target).GitHubRepository(), phase, branch), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("ok_select_branch_value_%s_%s_%s", target, phase, branch), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))

	closeAction := CloseButtonAction()
	return slack.NewBlockMessage(
		section,
		closeAction,
	)
}

func (h interactionHandler) SelectBranchInteraction(w http.ResponseWriter, interactionRequest slack.InteractionCallback) {
	actionValue := interactionRequest.ActionCallback.BlockActions[0].Value
	userID := interactionRequest.User.ID
	if !IsDeveploers(userID) {
		log.Print("[ERROR] Forbidden Error")
		responseBytes := getSlackError("Forbidden Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	arr := strings.Split(actionValue, "_")
	if len(arr) != 7 { // ok_select_branch_value_%s_%s_%s
		log.Print("[ERROR] Internal Server Error")
		responseBytes := getSlackError("Internal Server Error", "Please contact admin.", userID)
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
		return
	}

	res := h.deployApplication(arr[4], arr[5], arr[6])
	responseData := h.deployResponse(res, userID)
	responseData.ReplaceOriginal = true
	responseBytes, _ := json.Marshal(responseData)

	http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))

	return
}

func (h interactionHandler) deployApplication(target string, phase string, branch string) string {
	jobName := Project(target).JenkinsJob()
	url := fmt.Sprintf("https://bot:%s@%s/job/%s/buildWithParameters?token=%s&cause=slack-bot&ENV=%s&BRANCH=%s", h.jenkinsBotUserToken, os.Getenv("JENKINS_HOST"), jobName, h.jenkinsJobToken, phase, branch)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return err.Error()
	}
	if resp.StatusCode != 201 {
		return jobName + " Request failed. responsed " + fmt.Sprint(resp.StatusCode)
	}
	return fmt.Sprintf("Execute https://%s/job/%s/ \n selected branch: %s", os.Getenv("JENKINS_HOST"), jobName, branch)
}
