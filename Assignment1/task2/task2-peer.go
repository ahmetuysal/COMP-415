package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"net/rpc"
	"os"
)

const OPTIONS = "1) Enter the peer address to connect:\n2) Enter the key to find its successor:\n3) Enter the filename to take its hash:\n4) Display my-id, succ-id, and pred-id:\n5) Display the stored filenames and their keys:\n6) Exit."

type Peer struct {
	Id                 int32
	Address            string
	SuccessorId        *int32
	SuccessorAddress   *string
	PredecessorId      *int32
	PredecessorAddress *string
	FileIDs            []int32
	FileNames          map[int32]string
}

type PeerDTO struct {
	Id      int32
	Address string
}

func (peerInstance *Peer) FindSuccessor(id int32, peerDTO *PeerDTO) error {
	// if there is no successor, ring only consist of one peer and successor is the node itself
	if peerInstance.SuccessorId == nil {
		*peerDTO = PeerDTO{
			Id:      peerInstance.Id,
			Address: peerInstance.Address,
		}
	} else if *peerInstance.PredecessorId < id && id <= peerInstance.Id {
		*peerDTO = PeerDTO{
			Id:      peerInstance.Id,
			Address: peerInstance.Address,
		}
	} else if peerInstance.Id < id && id <= *peerInstance.SuccessorId {
		*peerDTO = PeerDTO{
			Id:      *peerInstance.SuccessorId,
			Address: *peerInstance.SuccessorAddress,
		}
	} else {
		// recursively call successor's FindSuccessor method via RPC
		successorOfPeerInstance, err := rpc.Dial("tcp", *peerInstance.SuccessorAddress) // connecting to the service
		if err != nil {
			return err
		}

		var successorOfId PeerDTO
		err = successorOfPeerInstance.Call("Peer.FindSuccessor", id, &successorOfId)
		if err != nil {
			return err
		}
		*peerDTO = successorOfId
	}

	return nil
}

func main() {
	port := os.Args[1]
	ip := getLocalIP()
	address := ip + ":" + port
	fmt.Println("Ip:", ip)
	fmt.Println("Port:", port)

	me := Peer{
		Id:        hashString(address),
		Address:   address,
		FileNames: make(map[int32]string),
	}

	_ = rpc.Register(&me)

	go listenForRpcConnections()
	go listenForTcpConnections()

	for {
		continue
	}
}


func listenForTcpConnections() {
	for {
		fmt.Println("Listening for TCP")
	}
}

func listenForRpcConnections() {
	for {
		fmt.Println("Listening for RPC")
	}
}

func hashString(value string) int32 {
	hashVal := sha256.Sum256([]byte(value))
	return int32(binary.BigEndian.Uint32(hashVal[:]))
}

func getLocalIP() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, address := range addresses {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	panic("Cannot get local ip address")
}
