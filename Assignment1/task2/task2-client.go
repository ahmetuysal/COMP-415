package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
)

const (
	PROMPT          = "1) Enter the filename to store:\n2) Enter the filename to retrieve:\n3) Exit:"
	CONNECTION_TYPE = "tcp"
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

func main() {
	serverIP := os.Args[1]
	port := os.Args[2]
	fmt.Println("Server Ip:" + serverIP + " port:" + port)
	address := serverIP + ":" + port
	connection, err := rpc.Dial(CONNECTION_TYPE, address)

	if err != nil {
		fmt.Println("Error when connecting to server on: "+serverIP+":"+port, err.Error())
		os.Exit(-1)
	}

	var selection int
	for {
		fmt.Println(PROMPT)
		_, _ = fmt.Scan(&selection)

		if selection == 1 {
			fmt.Println(">Enter the filename to store:")
			var filePath string
			_, _ = fmt.Scan(&filePath)

			file, err := os.Open(filePath)
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

			fileHash := hashString(fileInfo.Name())

			var successorOfFile PeerDTO
			_ = connection.Call("Peer.FindSuccessor", fileHash, &successorOfFile)

			fmt.Printf("%+x\n", successorOfFile)

			fileContent := make([]byte, fileInfo.Size())
			_, _ = file.Read(fileContent)
			_ = file.Close()

			fileDto := FileDTO{
				FileContent: fileContent,
				FileName:    fileInfo.Name(),
			}

			var fileConnection *rpc.Client
			if successorOfFile.Address == address {
				fileConnection = connection
			} else {
				fileConnection, err = rpc.Dial(CONNECTION_TYPE, successorOfFile.Address)
			}
			var isSuccessful bool
			_ = fileConnection.Call("Peer.ReceiveFile", fileDto, &isSuccessful)
			if successorOfFile.Address != address {
				_ = fileConnection.Close()
			}
			if isSuccessful {
				fmt.Printf(">Server Response: %s stored successfully.\n", fileInfo.Name())
			} else {
				fmt.Printf(">Server Response: Error occurred when storing %s.\n", fileInfo.Name())
			}
		} else if selection == 2 {
			fmt.Println(">Enter the filename to retrieve:")
			var fileName string
			_, _ = fmt.Scan(&fileName)

			fileHash := hashString(fileName)
			var successorOfFile PeerDTO
			err = connection.Call("Peer.FindSuccessor", fileHash, &successorOfFile)
			fmt.Println("After find successor")
			if err != nil {
				fmt.Println("Err:", err)
			}

			var fileConnection *rpc.Client
			if successorOfFile.Address == address {
				fileConnection = connection
			} else {
				fileConnection, err = rpc.Dial(CONNECTION_TYPE, successorOfFile.Address)
			}
			var fileDTO FileDTO

			err = fileConnection.Call("Peer.SendFile", fileName, &fileDTO)
			if successorOfFile.Address != address {
				_ = fileConnection.Close()
			}
			if err != nil {
				fmt.Println(">Server Response: File does not exist.")
			} else {
				fmt.Printf(">Server Response: %s found.\n", fileDTO.FileName)
				err = ioutil.WriteFile(fileDTO.FileName, fileDTO.FileContent, 0644)
				if err != nil {
					fmt.Println("Error occurred when saving the file")
				}
			}
		} else if selection == 3 {
			fmt.Println("Bye!")
			_ = connection.Close()
			return
		}
	}
}

func hashString(value string) uint32 {
	hashVal := sha256.Sum256([]byte(value))
	return binary.BigEndian.Uint32(hashVal[:])
}
