package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

// DOMAIN MODEL

// ChatMsg represents a message
type ChatMsg struct {
	Message string
	From    Peer
}

// Peer represents a other chat user and machine
type Peer struct {
	Name    string
	Address string
}

// Peers is the map of Peers with a address as key
type Peers map[string]Peer

// P2PSystem contains the complete p2p system
type P2PSystem struct {
	Self            Peer
	Peers           Peers
	receivedMsg     chan (ChatMsg)
	peerJoins       chan (Peer)
	peerLeft        chan (Peer)
	currentPeers    chan (Peers)
	getCurrentPeers chan (bool)
	userMsg         chan (ChatMsg)
}

// INITIALIZATION

func main() {
	port := flag.String("p", "8000", "Listen on port number")
	name := flag.String("n", "anonymous", "Nickname")
	peer := flag.String("j", "", "Other peer to join")
	flag.Parse()

	system := NewP2PSystem(Peer{*name, getLocalIpv4() + ":" + *port})
	system.start()
	if len(*peer) != 0 {
		system.peerJoins <- Peer{"", *peer}
	}

	system.startStdinListener(system.Self)
}

func getLocalIpv4() string {
	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			return fmt.Sprintf("%s", ipv4)
		}
	}
	return "localhost"
}

// HEART: P2P SYSTEM

// NewP2PSystem initializes a new P2PSystem and return a *P2PSystem
func NewP2PSystem(self Peer) *P2PSystem {
	system := new(P2PSystem)
	system.Self = self
	system.Peers = make(Peers)
	system.peerJoins = make(chan (Peer))
	system.currentPeers = make(chan (Peers))
	system.getCurrentPeers = make(chan (bool))

	system.userMsg = make(chan (ChatMsg))
	system.receivedMsg = make(chan (ChatMsg))
	return system
}

func (system *P2PSystem) start() {
	go system.selectLoop()
	go system.startWebListener()
	fmt.Printf("# \"%s\" listening on %s \n", system.Self.Name, system.Self.Address)
}

func (system *P2PSystem) selectLoop() {
	for {
		select {
		case peer := <-system.peerJoins:
			if !system.knownPeer(peer) {
				fmt.Printf("# Connected to: %s \n", peer.Address)
				system.Peers[peer.Address] = peer
				go system.sendJoin(peer)
			}

		case <-system.getCurrentPeers:
			system.currentPeers <- system.Peers

		case peer := <-system.peerLeft:
			delete(system.Peers, peer.Address)

		case chatMsg := <-system.receivedMsg:
			fmt.Printf("%s writes: %s\n", chatMsg.From.Name, chatMsg.Message)

		case chatMsg := <-system.userMsg:
			fmt.Printf("%s (self) says: %s\n", chatMsg.From.Name, chatMsg.Message)
			for _, peer := range system.Peers {
				go system.sendChat(peer, chatMsg)
			}
		}
	}
}

func (system *P2PSystem) knownPeer(peer Peer) bool {
	if peer.Address == system.Self.Address {
		return true
	}
	_, knownPeer := system.Peers[peer.Address]
	return knownPeer
}

// HTTP CLIENT : SENDING TO OTHER PEERS

func (system *P2PSystem) sendJoin(peer Peer) {
	URL := "http://" + peer.Address + "/join"

	qs, _ := json.Marshal(system.Self)

	resp, err := http.Post(URL, "application/json", bytes.NewBuffer(qs))
	if err != nil {
		system.peerLeft <- peer
		return
	}

	system.peerJoins <- peer

	defer resp.Body.Close()
	otherPeers := Peers{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&otherPeers)
	for _, peer := range otherPeers {
		system.peerJoins <- peer
	}
}

func (system *P2PSystem) sendChat(peer Peer, msg ChatMsg) {
	URL := "http://" + peer.Address + "/chat"

	qs, _ := json.Marshal(msg)

	_, err := http.Post(URL, "application/json", bytes.NewBuffer(qs))
	if err != nil {
		system.peerLeft <- peer
		return
	}
}

// HTTP SERVER : LISTENING TO OTHER PEERS

func (system *P2PSystem) startWebListener() {
	http.HandleFunc("/chat", createChatHandler(system))
	http.HandleFunc("/join", createJoinHandler(system))

	log.Fatal(http.ListenAndServe(system.Self.Address, nil))
}

func createJoinHandler(system *P2PSystem) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		joiner := Peer{}
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&joiner)
		if err != nil {
			log.Printf("Error unmarshalling from: %v", err)
		}

		system.peerJoins <- joiner

		system.getCurrentPeers <- true
		enc := json.NewEncoder(w)
		enc.Encode(<-system.currentPeers)
	}
}

func createChatHandler(system *P2PSystem) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cm := ChatMsg{}
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&cm)
		if err != nil {
			log.Printf("Error unmarshalling from: %v", err)
		}

		system.receivedMsg <- cm
	}
}

// LISTENER STANDARD IN : USER INPUT

func (system *P2PSystem) startStdinListener(sender Peer) {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, _ := reader.ReadString('\n')
		message := line[:len(line)-1]
		system.userMsg <- ChatMsg{message, sender}
	}
}
