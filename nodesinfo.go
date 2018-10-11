package main

import "time"

var PublicUpdaterService *NodeInfoUpdaterService

type NodeMessage struct {
	IP string `json:"ip"`
	NodeName string `json:"nodename"`
	NodesAvailableMap map[string]int `json:"nodesAvailableMap"`
}
type NodeInfoUpdaterService struct {
	NodeMessagesChannel chan *NodeMessage
	CuratorNodesMessagesStates map[string]*NodeMessage
	StatesMapIsLocked bool
}

func (nodeInfoUpdaterService *NodeInfoUpdaterService) load(){
	for{
		select{
		case c := <- nodeInfoUpdaterService.NodeMessagesChannel:
			go nodeInfoUpdaterService.SetState(c)
		}
	}
}

func (nodeInfoUpdaterService *NodeInfoUpdaterService) SetState(nodeMessage *NodeMessage){
	for nodeInfoUpdaterService.StatesMapIsLocked == true{
		time.Sleep(20*time.Millisecond)
	}
	nodeInfoUpdaterService.StatesMapIsLocked = true
	nodeInfoUpdaterService.CuratorNodesMessagesStates[nodeMessage.NodeName] = nodeMessage
	nodeInfoUpdaterService.StatesMapIsLocked = false
}

func (nodeInfoUpdaterService *NodeInfoUpdaterService) GetMessagesMapCopy() map[string]*NodeMessage{
	var messagesMapCopy = make(map[string]*NodeMessage)
	for nodeInfoUpdaterService.StatesMapIsLocked == true{
		time.Sleep(20*time.Millisecond)
	}
	nodeInfoUpdaterService.StatesMapIsLocked = true
	for key, value := range nodeInfoUpdaterService.CuratorNodesMessagesStates{
		messagesMapCopy[key] = value
	}
	nodeInfoUpdaterService.StatesMapIsLocked = false
	return messagesMapCopy
}
