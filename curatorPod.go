package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type CuratorPodSettings struct {
	IP string
	NodeName string
	kubernetesApiAddr string
	clusterAddr string
	authToken string
	clusterApiConnTimeout int
	registrationService *RegistrationService
	nodesInfoUpdaterService *NodeInfoUpdaterService
}

type NodePodSettings struct {
	IP string
	NodeName string
}

type NodePodConnection struct {
	ws *websocket.Conn
	PodConf *NodePodSettings
	PodIsAvailable bool
	PodNetworkChannel chan *networkMessage
}

func podLoader(config *Configuration) *CuratorPodSettings{
	var curatorPod CuratorPodSettings
	curatorPod.kubernetesApiAddr = config.KubernetesApiAddr
	curatorPod.clusterAddr = config.ClusterAddr
	curatorPod.authToken = config.AuthToken
	curatorPod.clusterApiConnTimeout = config.ClusterApiConnTimeout
	curatorPod.getPodIPEnv()
	curatorPod.getPodNode()
	curatorPod.	registrationService = &RegistrationService{
		false,
		make(map[string]*NodePodConnection),
		make(chan *CuratorConnectionData),
		make(chan string),
		make(map[string]string),
	}
	publicRegistrationServiceLink = curatorPod.registrationService
	go curatorPod.registrationService.load()
	go curatorPod.registrationService.PublicStateUpdates()
	curatorPod.nodesInfoUpdaterService = &NodeInfoUpdaterService{
		make(chan *NodeMessage),
		make(map[string]*NodeMessage),
		false,
	}
	PublicUpdaterService = curatorPod.nodesInfoUpdaterService
	go curatorPod.nodesInfoUpdaterService.load()
	go curatorPod.NodesRefreshScheduler()
	SlackAlertBotIsActive = false
	curatorPod.TryToLoadBots()
	curatorPod.refreshNodesMap()
	return &curatorPod
}

func (curatorPod *CuratorPodSettings) getPodIPEnv() {
	conn, err := net.Dial("udp", curatorPod.kubernetesApiAddr + ":443")
	checkErr(err)
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	curatorPod.IP = localAddr.IP.String()
}

func (curatorPod *CuratorPodSettings) refreshNodesMap(){
	message := map[string]interface{}{
		"pretty":"true",
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		Error.Fatalln(err)
	}
	req, err := http.NewRequest("GET", curatorPod.clusterAddr + "/api/v1/nodes", bytes.NewBuffer(bytesRepresentation))
	req.Header.Add("Authorization", "Bearer " + curatorPod.authToken)
	req.Header.Add("Accept", "application/json")
	tr := &http.Transport{
		IdleConnTimeout: 1000 * time.Millisecond * time.Duration(curatorPod.clusterApiConnTimeout),
		TLSHandshakeTimeout: 1000 * time.Millisecond * time.Duration(curatorPod.clusterApiConnTimeout),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport:tr}
	resp, err := client.Do(req)
	if err != nil {
		Error.Println("Error on response.\n[ERRO] -", err)
	}
	if resp != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		var nodesList NodeList
		if err := json.Unmarshal(body, &nodesList); err != nil {
			Warning.Println(err)
		}
		for _, node := range nodesList.Items{
			for _, state := range node.Status.Conditions{
				if state.Type == "Ready"{
					if state.Status == "True"{
						CuratorNodesStateMap[node.Metadata.Name] = 1
					} else {
						CuratorNodesStateMap[node.Metadata.Name] = 0
					}
				}
			}
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (curatorPod *CuratorPodSettings) getPodNode() {
	message := map[string]interface{}{
		"pretty":"true",
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		Error.Fatalln(err)
	}
	req, err := http.NewRequest("GET", curatorPod.clusterAddr + "/api/v1/pods", bytes.NewBuffer(bytesRepresentation))
	req.Header.Add("Authorization", "Bearer " + curatorPod.authToken)
	req.Header.Add("Accept", "application/json")
	tr := &http.Transport{
		IdleConnTimeout: 1000 * time.Millisecond * time.Duration(curatorPod.clusterApiConnTimeout),
		TLSHandshakeTimeout: 1000 * time.Millisecond * time.Duration(curatorPod.clusterApiConnTimeout),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport:tr}
	resp, err := client.Do(req)
	if err != nil {
		Error.Println("Error on response.\n[ERRO] -", err)
	}
	if resp != nil{
		body, _ := ioutil.ReadAll(resp.Body)
		var podsList PodList
		if err := json.Unmarshal(body, &podsList); err != nil {
			Warning.Println(err)
		}
		for _, pod := range podsList.Items{
			if pod.Metadata.Name == os.Getenv("HOSTNAME"){
				curatorPod.NodeName = pod.Spec.NodeName
				break
			}
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (curatorPod *CuratorPodSettings) NodesRefreshScheduler(){
	for {
		time.Sleep(5*time.Second)
		curatorPod.refreshNodesMap()
		curatorPod.ClientPodsNodesUpdate()
	}
}

func (curatorPod *CuratorPodSettings) ClientPodsNodesUpdate(){
	connectionMapCopy := publicRegistrationServiceLink.GetConnectionsMapCopy()
	if len(*connectionMapCopy) != 0 {
		var publicStateMapCopy = publicRegistrationServiceLink.PublicStateMap
		for _, nodeConn := range *connectionMapCopy{
			clientsIpMap, err := json.Marshal(publicStateMapCopy)
			checkErr(err)
			if nodeConn.PodIsAvailable == true {
				sendDataToClient(nodeConn.PodNetworkChannel, &networkMessage{
					"nodesInfo",
					string(clientsIpMap),
				})
			}
		}
	}
}

func (curatorPod *CuratorPodSettings) TryToLoadBots(){
	if os.Getenv("SLACK_BOT") == "1" &&
		os.Getenv("SLACK_CHANNEL") != "" &&
		os.Getenv("SLACK_USERNAME") != "" &&
		os.Getenv("SLACK_WEBHOOK_LINK") != "" {
		config := &SlackConfiguration{
			os.Getenv("SLACK_CHANNEL"),
			os.Getenv("SLACK_USERNAME"),
			os.Getenv("SLACK_WEBHOOK_LINK"),
		}
		getBot := SlackAlertBot{
			curatorPod,
			config.SlackChannel,
			config.Username,
			config.SlackWebHookLink,
			make(chan AlertMessage),
		}
		SlackAlertBotIsActive = true
		GlobalSlackAlertBot = &getBot
		go getBot.load()
	}
}

func getDateTime() string{
	Time := time.Now()
	dateTimeString := Time.Format("02.01.2006 15:04:05.") + strconv.Itoa(Time.Nanosecond())
	return dateTimeString
}

func checkTCPAvailable(addr string) bool{
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil{
		Warning.Println(addr, "is not available by TCP.")
		if conn != nil{
			conn.Close()
		}
		return false
	} else {
		if conn != nil{
			conn.Close()
		}
		return true
	}
}