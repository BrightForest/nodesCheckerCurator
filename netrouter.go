package main

import "encoding/json"

type netRouter struct {
	request *networkMessage
}

type networkMessage struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func (netRouter *netRouter) RequestsRouter(podConnect *NodePodConnection, request *networkMessage) {
	switch request.Action {
	case "ping":
		go netRouter.Ping(podConnect, request)
	case "registration":
		go netRouter.Registration(podConnect, request)
	case "nodeUpdate":
		netRouter.NodeUpdate(podConnect, request)
	case "alert":
		go netRouter.Alert(podConnect, request)
	default:
		netRouter.NotRecognized(podConnect)
	}
}

func (netRouter *netRouter) Ping(podConnect *NodePodConnection, message *networkMessage) {
	sendDataToClient(podConnect.PodNetworkChannel, &networkMessage{
		"pong",
		"",
	})
}

func (netRouter *netRouter) Registration(podConnect *NodePodConnection, message *networkMessage) {
	var podRegistrationInfo NodePodSettings
	err := json.Unmarshal([]byte(message.Content), &podRegistrationInfo)
	if err != nil {
		sendDataToClient(podConnect.PodNetworkChannel, &networkMessage{
			"registrationFailed",
			"Unable to read registration JSON"})
	} else {
		podConnect.PodIsAvailable = true
		podConnect.PodConf = &podRegistrationInfo
		connectionData := &CuratorConnectionData{
			podRegistrationInfo.NodeName,
			podConnect,
		}
		publicRegistrationServiceLink.RegisterConnChannel <- connectionData
		publicRegistrationServiceLink.PodStatesChannel <- &PodState{
			podRegistrationInfo.NodeName,
			1,
		}
		sendDataToClient(podConnect.PodNetworkChannel, &networkMessage{
			"registrationOk",
			"",
		})
		Info.Println(podRegistrationInfo.NodeName, "was registered")
	}
}

func (netRouter *netRouter) NodeUpdate(podConnect *NodePodConnection, message *networkMessage) {
	var nodeMessage NodeMessage
	err := json.Unmarshal([]byte(message.Content), &nodeMessage)
	checkErr(err)
	PublicUpdaterService.NodeMessagesChannel <- &nodeMessage
}

func (netRouter *netRouter) Alert(podConnect *NodePodConnection, message *networkMessage) {
	if CuratorNodesStateMap[podConnect.PodConf.NodeName] != 0 {
		var alertMessage AlertMessage
		err := json.Unmarshal([]byte(message.Content), &alertMessage)
		checkErr(err)
		if SlackAlertBotIsActive == true {
			GlobalSlackAlertBot.AlertChannel <- alertMessage
		}
		Warning.Println(alertMessage.NodeName, "was lost contact with", alertMessage.DisconnectedNodeName, "Time:", alertMessage.DateTime)
	}
}

func (netRouter *netRouter) NotRecognized(podConnect *NodePodConnection) {
	networkAnswer := &networkMessage{"error", "notRecognized"}
	sendDataToClient(podConnect.PodNetworkChannel, networkAnswer)
}

func sendDataToClient(podNetworkChannel chan *networkMessage, message *networkMessage) {
	podNetworkChannel <- message
}
