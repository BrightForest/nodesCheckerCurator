package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func (curatorPod *CuratorPodSettings) wsHandler(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize: 2048,
		WriteBufferSize: 2048,
	}
	ws, err := upgrader.Upgrade(w,r, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	pConn := &NodePodConnection{ws: ws, PodIsAvailable: false, PodNetworkChannel: make(chan *networkMessage, 10)}
	go curatorPod.getPodData(pConn)
}

func (curatorPod *CuratorPodSettings) getPodData(podConnect *NodePodConnection){
	requestsRouter := netRouter{};
	go curatorPod.writeServerDataLoop(podConnect)
	for{
		getMessage := networkMessage{}
		if podConnect.ws != nil{
			err := podConnect.ws.ReadJSON(&getMessage)
			if err != nil {
				Info.Println(podConnect.PodConf.NodeName, "was disconnected.")
				if podConnect.PodIsAvailable == true{
					podConnect.PodIsAvailable = false
					CuratorPodStatesMap[podConnect.PodConf.NodeName] = 0
					curatorPod.registrationService.UnregisterConnChannel <- podConnect.PodConf.NodeName
				}
				if podConnect.ws != nil{
					podConnect.ws.Close()
				}
				break
			}
			requestsRouter.request = &getMessage
			requestsRouter.RequestsRouter(podConnect, requestsRouter.request)
		} else {
			if podConnect.ws != nil{
				podConnect.ws.Close()
			}
			break
		}
	}
}

func (curatorPod *CuratorPodSettings) writeServerDataLoop(podConnect *NodePodConnection){
	for{
		myPodMessage := checkMyPodConnections(podConnect)
		if myPodMessage != nil{
//			Info.Println(myPodMessage.Action)
//			Info.Println(myPodMessage.Content)
			err := podConnect.ws.WriteJSON(myPodMessage)
			if err != nil{
				podConnect.PodIsAvailable = false
				Info.Println("Unable to write in WS Socket.")
				break
			}
		}
	}
}

func checkMyPodConnections(podConnect *NodePodConnection) *networkMessage{
	select {
		case message := <- podConnect.PodNetworkChannel:
			return message
	}
}

func (curatorPod *CuratorPodSettings) StartWSServer()  {
	Info.Println("Server was running.")
	http.HandleFunc("/ninf", curatorPod.ninf)
	http.HandleFunc("/nodesinfo", curatorPod.nodesInfo)
	http.HandleFunc("/podsinfo", curatorPod.podsInfo)
	http.HandleFunc("/podsipinfo", curatorPod.podsIpInfo)
	http.HandleFunc("/ws", curatorPod.wsHandler)
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go gracefulShutdownReciever(gracefulStop)
	if err := http.ListenAndServe(curatorPod.IP + ":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func (curatorPod *CuratorPodSettings) nodesInfo(w http.ResponseWriter, r *http.Request){
	for node, state := range CuratorNodesStateMap{
		fmt.Fprintln(w, "Node: " + node + " has state " + strconv.Itoa(state))
	}
}

func (curatorPod *CuratorPodSettings) podsInfo(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Registered pods")
	fmt.Fprintln(w, "--------------")
	var publicStateMapCopy = curatorPod.registrationService.PublicStateMap
	for podName, podIp := range publicStateMapCopy{
		fmt.Fprintln(w, "Pod: " + podName + " with IP " + podIp)
	}
}

func (curatorPod *CuratorPodSettings) podsIpInfo(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Pods and IP")
	fmt.Fprintln(w, "--------------")
	var publicStateMapCopy = curatorPod.registrationService.PublicStateMap
	for podName, podIp := range publicStateMapCopy{
		fmt.Fprintln(w, "Pod: " + podName + " with IP " + podIp)
	}
}

func (curatorPod *CuratorPodSettings) ninf(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Nodes visibles map:")
	fmt.Fprintln(w, "--------------")
	for node, lastMessage := range PublicUpdaterService.GetMessagesMapCopy(){
		nodesAvailableCount := 0
		for _, state := range lastMessage.NodesAvailableMap{
			if state == 1 {
				nodesAvailableCount++
			}
		}
		fmt.Fprintln(w, "Node: " + node + " [" + strconv.Itoa(nodesAvailableCount) + "/" + strconv.Itoa(len(CuratorNodesStateMap)) + "]")
		for nod, state := range lastMessage.NodesAvailableMap{
			fmt.Fprintln(w, "Visible: " + nod + " State: " + strconv.Itoa(state))
		}
		fmt.Fprintln(w, "--------------")
	}
}

func gracefulShutdownReciever(osChannel chan os.Signal){
	sig := <-osChannel
	Info.Printf("caught sig: %+v", sig)
	Info.Println("Wait for 2 second to finish processing")
	time.Sleep(2*time.Second)
	os.Exit(0)
}