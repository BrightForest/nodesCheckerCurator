package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"
)

var SlackAlertBotIsActive = false
var GlobalSlackAlertBot *SlackAlertBot

type AlertMessage struct {
	NodeName             string `json:"nodename"`
	DisconnectedNodeName string `json:"disconnectednode"`
	DateTime             string `json:"datetime"`
}

type SlackConfiguration struct {
	SlackChannel     string
	Username         string
	SlackWebHookLink string
}

type SlackAlertBot struct {
	CuratorPod       *CuratorPodSettings
	SlackChannel     string
	Username         string
	SlackWebHookLink string
	AlertChannel     chan AlertMessage
}

type SlackAlertMessage struct {
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	Icon_emoji string `json:"icon_emoji"`
	Text       string `json:"text"`
}

func (slackAlertBot *SlackAlertBot) load() {
	for {
		select {
		case message := <-slackAlertBot.AlertChannel:
			if checkTCPAvailable("hooks.slack.com:443") {
				go slackAlertBot.PushToSlack(message)
			} else {
				Error.Println("Unable to send Slack-Webhook - hooks.slack.com is not available.")
			}
		}
	}
}

func (slackAlertBot *SlackAlertBot) PushToSlack(alertMessage AlertMessage) {
	alertText := alertMessage.NodeName + " lost connection to " + alertMessage.DisconnectedNodeName + " at " + alertMessage.DateTime
	slackAlertMessage := SlackAlertMessage{
		slackAlertBot.SlackChannel,
		slackAlertBot.Username,
		":pepe:",
		alertText,
	}
	messageJson, err := json.Marshal(slackAlertMessage)
	checkErr(err)
	if err == nil {
		req, err := http.NewRequest("POST", slackAlertBot.SlackWebHookLink, bytes.NewBuffer(messageJson))
		checkErr(err)
		req.Header.Set("Content-Type", "application/json")

		tr := &http.Transport{
			IdleConnTimeout:     1000 * time.Millisecond * time.Duration(2),
			TLSHandshakeTimeout: 1000 * time.Millisecond * time.Duration(2),
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		checkErr(err)
		defer resp.Body.Close()
	}
}
