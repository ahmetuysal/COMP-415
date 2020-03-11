package main

import (
    "bufio"
    "fmt"
    "net"
    "net/rpc"
    "os"
)

const (
    DEFAULT_PORT    = "4444" // Use this default port if it isn't specified via command line arguments
    CONNECTION_TYPE = "tcp"
)

type MessengerProcess struct {
    OID                 string
    QueuedMessages      []MessageDTO
    VectorClock         []int
    VectorClockIndexMap map[string]int
}

type MessageDTO struct {
    Transcript string
    OID        string
    TimeStamp  []int
}

// Returns whether this message can be immediately delivered according to causally ordered multicast algorithm, i.e,
// it returns true if (message.TimeStamp[i] = VectorClock[i] + 1 for i = index of original sender id in vector clock
//				      and message.TimeStamp[k] <= VectorClock[k] for all k != i)
// it returns false otherwise
func (messengerProcess *MessengerProcess) shouldDeliverMessageToApplication(message MessageDTO) bool {
    i := messengerProcess.VectorClockIndexMap[message.OID]
    if message.TimeStamp[i] != messengerProcess.VectorClock[i]+1 {
        return false
    }

    for k, time := range messengerProcess.VectorClock {
        if k == i {
            continue
        }
        if message.TimeStamp[k] > time {
            return false
        }
    }

    return true
}

func (messengerProcess *MessengerProcess) deliverMessageToApplication(message MessageDTO) {
    // increase the time of this messenger process since it received a message
    i := messengerProcess.VectorClockIndexMap[messengerProcess.OID]
    messengerProcess.VectorClock[i]++
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

func (messengerProcess *MessengerProcess) PostMessage(message MessageDTO, isSuccessful *bool) error {
    fmt.Printf("%+x\n", message)
    if messengerProcess.shouldDeliverMessageToApplication(message) {
        messengerProcess.deliverMessageToApplication(message)
        // if a message is delivered vector clock is updated and we need to check queued messages
        for len(messengerProcess.QueuedMessages) > 0 {
            var nextMessageToDeliver MessageDTO
            var nextMessageToDeliverIndex int
            for index, queuedMessage := range messengerProcess.QueuedMessages {
                if messengerProcess.shouldDeliverMessageToApplication(queuedMessage) {
                    nextMessageToDeliver = queuedMessage
                    nextMessageToDeliverIndex = index
                    break
                }
            }

            // no message satisfies the condition
            if nextMessageToDeliver.OID == "" {
                break
            }
            messengerProcess.deliverMessageToApplication(nextMessageToDeliver)
            messengerProcess.QueuedMessages = append(messengerProcess.QueuedMessages[:nextMessageToDeliverIndex],
                messengerProcess.QueuedMessages[nextMessageToDeliverIndex+1:]...)
        }
    } else {
        // add message to queue since it does not satisfy the condition to deliver
        messengerProcess.QueuedMessages = append(messengerProcess.QueuedMessages, message)
    }
    *isSuccessful = true
    return nil
}

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
    peersFile, err := os.Open(peerFileName)
    handleError(err)
    defer peersFile.Close()

    peerToIndexMap := make(map[string]int)
    scanner := bufio.NewScanner(peersFile)
    for i := 0; scanner.Scan(); i++ {
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
        panic("Peers file does not include my address!")
    }

    me := MessengerProcess{
        OID:                 myAddress,
        QueuedMessages:      nil,
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
        fmt.Println(me.VectorClock)
    }
}
