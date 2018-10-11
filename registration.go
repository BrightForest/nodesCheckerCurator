package main

import "time"

var publicRegistrationServiceLink *RegistrationService

var CuratorPodStatesMap = make(map[string]int)
var CuratorNodesStateMap = make(map[string]int)

type CuratorConnectionData struct {
	NodeName string
	Connection *NodePodConnection
}

type RegistrationService struct {
	RegUpdateState bool
	ConnectionsMap map[string]*NodePodConnection
	RegisterConnChannel chan *CuratorConnectionData
	UnregisterConnChannel chan string
	PublicStateMap map[string]string
}

func (registrationService *RegistrationService) load(){
	for {
		select{
			case c:= <-registrationService.RegisterConnChannel:
				if !registrationService.connectionRegistration(c){
					Error.Println("Unavailable to register pod due map concurrent access error.")
				}
			case c:= <-registrationService.UnregisterConnChannel:
				if !registrationService.connectionUnregister(c) {
					Error.Println("Unavailable to unregister pod due map concurrent access error.")
				}
		}
	}
}

func (registrationService *RegistrationService) PublicStateUpdates(){
	for{
		time.Sleep(1*time.Second)
		for registrationService.RegUpdateState == true{
			time.Sleep(20*time.Millisecond)
		}
		var updatePublicStateMap = make(map[string]string)
		registrationService.RegUpdateState = true
		for podName, podConn := range registrationService.ConnectionsMap{
			updatePublicStateMap[podName] = podConn.PodConf.IP}
		registrationService.PublicStateMap = updatePublicStateMap
		updatePublicStateMap = nil
		registrationService.RegUpdateState = false}
}

func (registrationService *RegistrationService) connectionRegistration(connectionData *CuratorConnectionData) bool{
	for registrationService.RegUpdateState == true {
		time.Sleep(20*time.Millisecond)
	}
	registrationService.RegUpdateState = true
	registrationService.ConnectionsMap[connectionData.NodeName] = connectionData.Connection
	registrationService.RegUpdateState = false
	return true
}

func (registrationService *RegistrationService) connectionUnregister(nodeName string) bool{
	for registrationService.RegUpdateState == true {
		time.Sleep(20*time.Millisecond)
	}
	registrationService.RegUpdateState = true
	delete(registrationService.ConnectionsMap, nodeName)
	registrationService.RegUpdateState = false
	return true
}

func (registrationService *RegistrationService) GetConnectionsMapCopy() *map[string]*NodePodConnection{
	var connMapCopy = make(map[string]*NodePodConnection)
	for registrationService.RegUpdateState == true {
		time.Sleep(20*time.Millisecond)
	}
	registrationService.RegUpdateState = true
	for nodeName, nodeConn := range registrationService.ConnectionsMap{
		connMapCopy[nodeName] = nodeConn
	}
	registrationService.RegUpdateState = false
	return &connMapCopy
}