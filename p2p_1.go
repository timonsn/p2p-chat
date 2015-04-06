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

type ChatMsg struct {
	Message string
	From    Peer
}

type Peer struct {
	Name    string
	Address string
}

type Peers map[string]Peer

type P2PSystem struct {
	Self            Peer
	Peers           Peers
	peerJoins       chan (Peer)
	peerLeft        chan (Peer)
	currentPeers    chan (Peers)
	getCurrentPeers chan (bool)
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

func NewP2PSystem(self Peer) *P2PSystem {
	system := new(P2PSystem)
	system.Self = self
	system.Peers = make(Peers)
	system.peerJoins = make(chan (Peer))
	system.currentPeers = make(chan (Peers))
	system.getCurrentPeers = make(chan (bool))

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
	finalUrl := "http://" + peer.Address + "/join"

	qs, _ := json.Marshal(system.Self)

	resp, err := http.Post(finalUrl, "application/json", bytes.NewBuffer(qs))
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

// HTTP SERVER : LISTENING TO OTHER PEERS

func (system *P2PSystem) startWebListener() {
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

// LISTENER STANDARD IN : USER INPUT

func (system *P2PSystem) startStdinListener(sender Peer) {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, _ := reader.ReadString('\n')
		message := line[:len(line)-1]

		//do something with message
	}
}
