package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
)

const (
	DEFAULT_PORT = "4444" // Use this default port if it isn't specified via command line arguments
	CONNECTION_TYPE = "tcp"
)

type MessengerProcess struct {
	vectorClock []int
}

type MessageDTO struct {
	Transcript string
	OID string
	TimeStamp []int
}


func (messengerProcess *MessengerProcess) MessagePost(message *MessageDTO, isSuccessful *bool) error {
	return nil
}

func handleError(e error) {
	if e != nil {
		panic(e)
	}
}

func checkPeersForStart(peers map[string]int, myAddress string) {
	fmt.Println("Welcome to Decentralized Group Messenger, please wait until all other peers are available.")
	peerAddresses := make([]string, len(peers))
	for peerAddress, index := range peers {
		peerAddresses[index] = peerAddress
	}

	peerAvailable := make([]bool, len(peers))

	remainingPeers := len(peers)
	for remainingPeers > 0 {
		for index, isAlreadyResponded := range peerAvailable {
			// we have already reached this peer
			if isAlreadyResponded {
				continue
			}

			// this peer is us :)
			if peerAddresses[index] == myAddress {
				peerAvailable[index] = true
				remainingPeers--
				continue
			}

			rpcConnection, err := rpc.Dial(CONNECTION_TYPE, peerAddresses[index])
			if err != nil {
				fmt.Println(err)
				continue
			} else {
				_ = rpcConnection.Close()
				peerAvailable[index] = true
				remainingPeers--
			}
		}
	}
	fmt.Println("All peers are available, you can start sending messages")
}


func readPeersAndInitializeVectorClock(peerFileName string) (map[string]int, []int) {
	peersFile, err := os.Open("peers.txt")
	handleError(err)
	defer peersFile.Close()

	peerToIndexMap := make(map[string]int)
	scanner := bufio.NewScanner(peersFile)
	for i := 0; scanner.Scan() ; i++ {
		peerToIndexMap[scanner.Text()] = i
	}
	initialVectorClock := make([]int, len(peerToIndexMap))
	return peerToIndexMap, initialVectorClock
}

func getLocalAddress() string {
	port := DEFAULT_PORT
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	conn, err := net.Dial("udp", "eng.ku.edu.tr:80")
	handleError(err)
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String() + ":" + port
}

func multicastMessage(transcript string, currentClock *[]int, peers map[string]int, myId string) {
	myIndex := peers[myId]
	(*currentClock)[myIndex]++
	cloneVectorClock :=append((*currentClock)[:0:0], *currentClock...)
	message := MessageDTO{
		Transcript: transcript,
		OID:        myId,
		TimeStamp:  cloneVectorClock,
	}

	fmt.Printf("%+v\n", message)

	for peer, index := range peers {
		if peer == myId {
			fmt.Println(peer)
			continue
		}
		fmt.Println(index)
	}

 }


func listenForRpcConnections(listener *net.TCPListener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		rpc.ServeConn(conn)
	}
}


func main() {
	myAddress := getLocalAddress()
	peers, vectorClock := readPeersAndInitializeVectorClock("peers.txt")

	tcpAddr, _ := net.ResolveTCPAddr(CONNECTION_TYPE, myAddress)
	listener, _ := net.ListenTCP(CONNECTION_TYPE, tcpAddr)
	go listenForRpcConnections(listener)

	checkPeersForStart(peers, myAddress)

	multicastMessage("Hello, world!", &vectorClock, peers, myAddress)

	fmt.Println(vectorClock)

	for {
		continue
	}
}
