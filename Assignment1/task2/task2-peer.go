package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"strings"
)

const (
	OPTIONS         = "1) Enter the peer address to connect:\n2) Enter the key to find its successor:\n3) Enter the filename to take its hash:\n4) Display my-id, succ-id, and pred-id:\n5) Display the stored filenames and their keys:\n6) Exit."
	CONNECTION_TYPE = "tcp"
	BUFFER_SIZE     = 65495
)

type Peer struct {
	Id                 uint32
	Address            string
	SuccessorId        *uint32
	SuccessorAddress   *string
	PredecessorId      *uint32
	PredecessorAddress *string
	// filename contains both user and file name, e.g, ahmet/task2-peer.go
	FileNames map[uint32]string
}

type PeerDTO struct {
	Id      uint32
	Address string
}

type FileDTO struct {
	FileContent []byte
	FileName    string
}

func (peerInstance *Peer) FindSuccessor(id uint32, peerDTO *PeerDTO) error {
	fmt.Printf("FindSuccessor %d\n", id)
	// fmt.Printf("%+v\n", peerInstance)
	// if there is no successor, ring only consist of one peer and successor is the node itself
	if peerInstance.SuccessorId == nil {
		*peerDTO = PeerDTO{
			Id:      peerInstance.Id,
			Address: peerInstance.Address,
		}
	} else if (*peerInstance.PredecessorId < id || peerInstance.Id < *peerInstance.PredecessorId) && id <= peerInstance.Id {
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
		successorOfPeerInstance, err := rpc.Dial(CONNECTION_TYPE, *peerInstance.SuccessorAddress) // connecting to the service
		if err != nil {
			return err
		}

		var successorOfId PeerDTO
		err = successorOfPeerInstance.Call("Peer.FindSuccessor", id, &successorOfId)
		if err != nil {
			return err
		}
		err = successorOfPeerInstance.Close()
		if err != nil {
			return err
		}
		*peerDTO = successorOfId
	}

	fmt.Println("Find Successor last line")

	return nil
}

func (peerInstance *Peer) SetPredecessor(peerDTO *PeerDTO, exPredecessor *PeerDTO) error {
	fmt.Printf("Set Predecessor %+x\n", peerDTO)
	if peerInstance.PredecessorId != nil {
		*exPredecessor = PeerDTO{
			Id:      *peerInstance.PredecessorId,
			Address: *peerInstance.PredecessorAddress,
		}
	}
	peerInstance.PredecessorId = &peerDTO.Id
	peerInstance.PredecessorAddress = &peerDTO.Address

	predecessorRpcConnection, err := rpc.Dial(CONNECTION_TYPE, peerDTO.Address)
	if err != nil {
		return err
	}

	for fileHash, filename := range peerInstance.FileNames {
		// TODO: validate this condition
		if (peerInstance.Id > peerDTO.Id && (fileHash > peerInstance.Id || fileHash <= peerDTO.Id)) || (peerDTO.Id > peerInstance.Id && fileHash > peerInstance.Id) {
			file, err := os.Open(filename)
			if err != nil {
				fmt.Println("The file can't be opened", err)
				continue
			}

			fileInfo, err := file.Stat()
			if err != nil {
				fmt.Println("Can't get the file info", err)
				_ = file.Close()
				continue
			}

			fileContent := make([]byte, fileInfo.Size())
			_, _ = file.Read(fileContent)

			var isSuccessful bool
			_ = predecessorRpcConnection.Call("Peer.ReceiveFile", FileDTO{
				FileContent: fileContent,
				FileName:    filename,
			}, &isSuccessful)
		}
	}
	_ = predecessorRpcConnection.Close()

	return nil
}

func (peerInstance *Peer) SetSuccessor(peerDTO *PeerDTO, exSuccessor *PeerDTO) error {
	fmt.Printf("Set Successor %+x\n", peerDTO)
	if peerInstance.SuccessorId != nil {
		*exSuccessor = PeerDTO{
			Id:      *peerInstance.SuccessorId,
			Address: *peerInstance.SuccessorAddress,
		}
	}
	peerInstance.SuccessorId = &peerDTO.Id
	peerInstance.SuccessorAddress = &peerDTO.Address
	return nil
}

func (peerInstance *Peer) ReceiveFile(fileDTO *FileDTO, isSuccessful *bool) error {
	fmt.Println("Receiving file", fileDTO.FileName)
	err := ioutil.WriteFile(fileDTO.FileName, fileDTO.FileContent, 0644)
	if err != nil {
		*isSuccessful = false
		return err
	}
	peerInstance.FileNames[hashString(fileDTO.FileName)] = fileDTO.FileName
	*isSuccessful = true
	return nil
}

func (peerInstance *Peer) SendFile(fileName string , fileDTO *FileDTO) error {
	fmt.Printf("Send File %s\n", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	fileContent := make([]byte, fileInfo.Size())
	_, _ = file.Read(fileContent)
	fileDTO.FileName = fileInfo.Name()
	fileDTO.FileContent = fileContent
	return nil
}


func main() {
	port := os.Args[1]
	conn, _ := net.Dial(CONNECTION_TYPE, "github.com:80")
	_ = conn.Close()
	ip := strings.Split(conn.LocalAddr().String(), ":")[0]

	address := ip + ":" + port
	fmt.Println("Ip:", ip)
	fmt.Println("Port:", port)

	me := Peer{
		Id:        hashString(address),
		Address:   address,
		FileNames: make(map[uint32]string),
	}

	_ = rpc.Register(&me)

	tcpAddr, _ := net.ResolveTCPAddr(CONNECTION_TYPE, ":"+port)
	listener, _ := net.ListenTCP(CONNECTION_TYPE, tcpAddr)
	go listenForRpcConnections(listener)

	/*
		Message format:
		1st character = 0 if peer-peer 1 if peer-client interaction
		2nd character denotes message type
		rest of the message is payload
	*/

	var selection int
	for {
		fmt.Println(OPTIONS)
		_, _ = fmt.Scan(&selection)

		switch selection {
		case 1:
			var peerAddress string
			fmt.Print(">Enter the peer address to connect:")
			_, _ = fmt.Scan(&peerAddress)
			rpcConnection, err := rpc.Dial(CONNECTION_TYPE, peerAddress)
			if err != nil {
				fmt.Println("Error when connecting to peer on: "+peerAddress, err.Error())
				continue
			}

			// find successor to register
			var successorDTO PeerDTO
			_ = rpcConnection.Call("Peer.FindSuccessor", me.Id, &successorDTO)
			_ = rpcConnection.Close()
			me.SuccessorId = &successorDTO.Id
			me.SuccessorAddress = &successorDTO.Address

			successorRpcConnection, err := rpc.Dial(CONNECTION_TYPE, *me.SuccessorAddress)
			if err != nil {
				fmt.Println("Error when connecting to peer on: "+peerAddress, err.Error())
				continue
			}

			var successorsExPredecessor PeerDTO
			_ = successorRpcConnection.Call("Peer.SetPredecessor", PeerDTO{
				Id:      me.Id,
				Address: me.Address,
			}, &successorsExPredecessor)
			_ = successorRpcConnection.Close()

			// successor doesn't have an ex predecessor
			if successorsExPredecessor.Address == "" {
				me.PredecessorId = &successorDTO.Id
				me.PredecessorAddress = &successorDTO.Address
			} else {
				me.PredecessorId = &successorsExPredecessor.Id
				me.PredecessorAddress = &successorsExPredecessor.Address
			}

			predecessorRpcConnection, err := rpc.Dial(CONNECTION_TYPE, *me.PredecessorAddress)
			if err != nil {
				fmt.Println("Error when connecting to peer on: "+peerAddress, err.Error())
				continue
			}

			var dummyDto PeerDTO
			_ = predecessorRpcConnection.Call("Peer.SetSuccessor", PeerDTO{
				Id:      me.Id,
				Address: me.Address,
			}, &dummyDto)
			_ = predecessorRpcConnection.Close()

			// Note: responsibility of moving files is delegated to SetSuccessor method

			fmt.Println(">(Response): Connection Established")
		case 2:
			var key uint32
			fmt.Print(">Enter the key to find its successor: ")
			_, _ = fmt.Scan(&key)
			var successor PeerDTO
			_ = me.FindSuccessor(key, &successor)
			successorIPAndPort := strings.Split(successor.Address, ":")
			fmt.Printf(">(Response): (%d, %s, %s)\n", successor.Id, successorIPAndPort[0], successorIPAndPort[1])
		case 3:
			var fileName string
			fmt.Print("3) Enter the filename to take its hash: ")
			_, _ = fmt.Scan(&fileName)
			fileNameHash := hashString(fileName)
			fmt.Println(">(Response):", fileNameHash)
		case 4:
			if me.SuccessorId == nil {
				fmt.Println("my-id:", me.Id, "succ-id:", me.SuccessorId, "pred-id:", me.PredecessorId)
			} else {
				fmt.Println("my-id:", me.Id, "succ-id:", *me.SuccessorId, "pred-id:", *me.PredecessorId)
			}
		case 5:
			fmt.Println("Stored keys and their associated file names:", me.FileNames)
		case 6:
			// if peer is connected to a ring, update succ and pred
			if me.PredecessorId != nil {
				var dummyDto PeerDTO

				successorRpcConnection, err := rpc.Dial(CONNECTION_TYPE, *me.SuccessorAddress)
				if err != nil {
					fmt.Println("Error when connecting to peer on: "+*me.SuccessorAddress, err.Error())
				}
				_ = successorRpcConnection.Call("Peer.SetPredecessor", PeerDTO{
					Id:      *me.PredecessorId,
					Address: *me.PredecessorAddress,
				}, &dummyDto)

				predecessorRpcConnection, err := rpc.Dial(CONNECTION_TYPE, *me.PredecessorAddress)
				if err != nil {
					fmt.Println("Error when connecting to peer on: "+*me.PredecessorAddress, err.Error())
					continue
				}

				_ = predecessorRpcConnection.Call("Peer.SetSuccessor", PeerDTO{
					Id:      *me.SuccessorId,
					Address: *me.SuccessorAddress,
				}, &dummyDto)

				for _, filename := range me.FileNames {
					file, err := os.Open(filename)
					if err != nil {
						fmt.Println("The file can't be opened", err)
						continue
					}

					fileInfo, err := file.Stat()
					if err != nil {
						fmt.Println("Can't get the file info", err)
						_ = file.Close()
						continue
					}

					fileContent := make([]byte, fileInfo.Size())
					_, _ = file.Read(fileContent)

					var isSuccessful bool
					_ = predecessorRpcConnection.Call("Peer.ReceiveFile", FileDTO{
						FileContent: fileContent,
						FileName:    fileInfo.Name(),
					}, &isSuccessful)
				}
				_ = predecessorRpcConnection.Close()
			}
			fmt.Println("Bye!")
			return
		}
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

func hashString(value string) uint32 {
	hashVal := sha256.Sum256([]byte(value))
	return binary.BigEndian.Uint32(hashVal[:])
}
