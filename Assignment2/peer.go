package main

import (
    "bufio"
    "fmt"
    "net"
    "net/rpc"
    "os"
    "sync"
)

const (
    DEFAULT_PORT    = "4444" // Use this default port if it isn't specified via command line arguments
    CONNECTION_TYPE = "tcp"
)

type MessageDTO struct {
    Transcript string
    OID        string
    TimeStamp  []int
}

// Returns whether this message can be immediately delivered according to causally ordered multicast algorithm, i.e,
// it returns true if (message.TimeStamp[i] = VectorClock[i] + 1 for i = index of original sender id in vector clock
//				      and message.TimeStamp[k] <= VectorClock[k] for all k != i)
// it returns false otherwise
func (message MessageDTO) shouldDeliverMessageToApplication(vectorClock []int, senderProcessIndex int) bool {
    if message.TimeStamp[senderProcessIndex] != vectorClock[senderProcessIndex]+1 {
        return false
    }

    for k, time := range vectorClock {
        if k == senderProcessIndex {
            continue
        }
        if message.TimeStamp[k] > time {
            return false
        }
    }

    return true
}

// Concurrent Message Slice Implementation
// Created by following https://dnaeon.github.io/concurrent-maps-and-slices-in-go/
type ConcurrentMessageSlice struct {
    sync.RWMutex
    messages []MessageDTO
}

func (cms *ConcurrentMessageSlice) append(message MessageDTO) {
    cms.Lock()
    defer cms.Unlock()
    cms.messages = append(cms.messages, message)
}

// Note: Removal of messages from the slice is implemented in MessengerProcess struct

// End of Concurrent Message Slice Implementation

type MessengerProcess struct {
    OID                 string
    QueuedMessages      ConcurrentMessageSlice
    VectorClock         []int
    VectorClockIndexMap map[string]int
}

// Iterates all queued messages of the process and returns the first one who satisfies the message delivery conditions
// returns an empty MessageDTO if there are no messages that satisfies the conditions
func (messengerProcess *MessengerProcess) popNextMessageToDeliver() (MessageDTO, bool) {
    var nextMessageToDeliver MessageDTO
    var nextMessageToDeliverIndex int
    messengerProcess.QueuedMessages.Lock()
    defer messengerProcess.QueuedMessages.Unlock()

    for index, queuedMessage := range messengerProcess.QueuedMessages.messages {
        if messengerProcess.shouldDeliverMessageToApplication(queuedMessage) {
            nextMessageToDeliver = queuedMessage
            nextMessageToDeliverIndex = index
            break
        }
    }

    // no message satisfies the condition
    if nextMessageToDeliver.OID == "" {
        return MessageDTO{}, false
    }
    messengerProcess.deliverMessageToApplication(nextMessageToDeliver)
    messengerProcess.QueuedMessages.messages = append(messengerProcess.QueuedMessages.messages[:nextMessageToDeliverIndex],
        messengerProcess.QueuedMessages.messages[nextMessageToDeliverIndex+1:]...)
    return nextMessageToDeliver, true
}

// Returns whether this message can be immediately delivered according to causally ordered multicast algorithm, i.e,
// it returns true if (message.TimeStamp[i] = VectorClock[i] + 1 for i = index of original sender id in vector clock
//				      and message.TimeStamp[k] <= VectorClock[k] for all k != i)
// it returns false otherwise
func (messengerProcess MessengerProcess) shouldDeliverMessageToApplication(message MessageDTO) bool {
    i := messengerProcess.VectorClockIndexMap[message.OID]
    return message.shouldDeliverMessageToApplication(messengerProcess.VectorClock, i)
}

// Updates the vector clock of the process according to distributed multicast algorithm and prints the sender and message
// transcript to standard out
func (messengerProcess *MessengerProcess) deliverMessageToApplication(message MessageDTO) {
    // update the vector clock
    for index, time := range messengerProcess.VectorClock {
        var maxTime int
        if time > message.TimeStamp[index] {
            maxTime = time
        } else {
            maxTime = message.TimeStamp[index]
        }
        messengerProcess.VectorClock[index] = maxTime
    }
    // print the message on console
    fmt.Printf("%s: %s\n", message.OID, message.Transcript)
}

// RPC function to send a message to a peer
func (messengerProcess *MessengerProcess) PostMessage(message MessageDTO, isSuccessful *bool) error {
    fmt.Printf("%+x\n", message)
    if messengerProcess.shouldDeliverMessageToApplication(message) {
        messengerProcess.deliverMessageToApplication(message)
        // if a message is delivered vector clock is updated and we need to check queued messages
        nextMessageToDeliver, shouldDeliver := messengerProcess.popNextMessageToDeliver()
        for shouldDeliver {
            messengerProcess.deliverMessageToApplication(nextMessageToDeliver)
            nextMessageToDeliver, shouldDeliver = messengerProcess.popNextMessageToDeliver()
        }
    } else {
        // add message to queue since it does not satisfy the condition to deliver
        messengerProcess.QueuedMessages.append(message)
    }
    *isSuccessful = true
    return nil
}

// Iterates over all peers and posts the message via RPC calls
func (messengerProcess *MessengerProcess) multicastMessage(transcript string) {
    i := messengerProcess.VectorClockIndexMap[messengerProcess.OID]
    messengerProcess.VectorClock[i]++
    cloneVectorClock := append((messengerProcess.VectorClock)[:0:0], messengerProcess.VectorClock...)
    message := MessageDTO{
        Transcript: transcript,
        OID:        messengerProcess.OID,
        TimeStamp:  cloneVectorClock,
    }

    for peerAddress := range messengerProcess.VectorClockIndexMap {
        if peerAddress == messengerProcess.OID {
            continue
        }
        go postMessageToPeer(peerAddress, message)
    }
}

// posts message to the peer at given address with an RPC call
func postMessageToPeer(peerAddress string, message MessageDTO) {
    peerRpcConnection, err := rpc.Dial(CONNECTION_TYPE, peerAddress)
    handleError(err)
    var isSuccessful bool
    _ = peerRpcConnection.Call("MessengerProcess.PostMessage", message, &isSuccessful)
    _ = peerRpcConnection.Close()
}

func handleError(e error) {
    if e != nil {
        panic(e)
    }
}

// checks whether all peers are ready by trying to connect to all peers until everyone is available
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

// reads the peer file and returns a peerToIndexMap and a initialVectorClock
func readPeersAndInitializeVectorClock(peerFileName string) (map[string]int, []int) {
    peersFile, err := os.Open(peerFileName)
    handleError(err)
    defer func() {
        handleError(peersFile.Close())
    }()

    peerToIndexMap := make(map[string]int)
    scanner := bufio.NewScanner(peersFile)
    for i := 0; scanner.Scan(); i++ {
        peerToIndexMap[scanner.Text()] = i
    }
    initialVectorClock := make([]int, len(peerToIndexMap))
    return peerToIndexMap, initialVectorClock
}

// returns the local address of this device
func getLocalAddress() string {
    port := DEFAULT_PORT
    if len(os.Args) > 1 {
        port = os.Args[1]
    }
    conn, err := net.Dial("udp", "eng.ku.edu.tr:80")
    handleError(err)
    defer func() {
        handleError(conn.Close())
    }()

    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String() + ":" + port
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

    _, ok := peers[myAddress]

    if !ok {
        panic("Peers file does not include my address: " + myAddress)
    }

    me := MessengerProcess{
        OID: myAddress,
        QueuedMessages: ConcurrentMessageSlice{
            RWMutex:  sync.RWMutex{},
            messages: nil,
        },
        VectorClock:         vectorClock,
        VectorClockIndexMap: peers,
    }

    _ = rpc.Register(&me)

    tcpAddr, _ := net.ResolveTCPAddr(CONNECTION_TYPE, myAddress)
    listener, _ := net.ListenTCP(CONNECTION_TYPE, tcpAddr)
    go listenForRpcConnections(listener)

    checkPeersForStart(peers, myAddress)
    scanner := bufio.NewScanner(os.Stdin)
    var transcript string
    fmt.Print("> Enter a message to send: ")
    for scanner.Scan() {
        transcript = scanner.Text()
        me.multicastMessage(transcript)
        fmt.Print("> Enter a message to send: ")
    }
}
