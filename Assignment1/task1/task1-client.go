package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	BUFFER_SIZE     = 65495
	CONNECTION_TYPE = "tcp"
)

func main() {
	serverIP := os.Args[1]
	port := os.Args[2]
	fmt.Println("Server Ip:" + serverIP + " port:" + port)

	connection, err := net.Dial(CONNECTION_TYPE, serverIP+":"+port)

	if err != nil {
		fmt.Println("Error when connecting to server on: "+serverIP+":"+port, err.Error())
		os.Exit(-1)

	}

	var selection int
	for {
		fmt.Println("1) Enter the username:\n2) Enter the filename to store:\n3) Enter the filename to retrieve:\n4) Exit:")
		_, _ = fmt.Scan(&selection)
		var serverResponse string
		if selection == 1 {
			var username string
			fmt.Print(">Enter the username: ")
			_, _ = fmt.Scan(&username)
			_, _ = fmt.Fprintf(connection, "1"+username+"\n")
		} else if selection == 2 {
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

			storeStartTime := time.Now()
			fileSize := strconv.FormatInt(fileInfo.Size(), 10)
			fileName := fileInfo.Name()
			_, _ = fmt.Fprintf(connection, "2"+fileSize+":"+fileName+"\n")
			sendFile(connection, file)
			_ = file.Close()
			storeFinishTime := time.Since(storeStartTime)
			fmt.Printf("%d KBs stored in %f seconds\n", fileInfo.Size()/1024.0, storeFinishTime.Seconds())

		} else if selection == 3 {
			fmt.Println(">Enter the filename to retrieve:")
			var fileName string
			_, _ = fmt.Scan(&fileName)

			retrieveStartTime := time.Now()

			_, _ = fmt.Fprintf(connection, "3"+fileName+"\n")
			serverResponse, _ = bufio.NewReader(connection).ReadString('\n')

			if serverResponse[0] == 51 {
				separatorIndex := strings.Index(serverResponse, ":")
				fileSize, _ := strconv.ParseInt(serverResponse[1:separatorIndex], 10, 64)
				fileName := serverResponse[separatorIndex+1 : len(serverResponse)-1]
				_ = receiveFile(connection, fileName, fileSize)
				retrieveEndTime := time.Since(retrieveStartTime)
				fmt.Printf("%d KBs retrieved in %f seconds\n", fileSize/1024.0, retrieveEndTime.Seconds())
			} else {
				fmt.Print("Server Response: " + serverResponse)
				continue
			}
		} else if selection == 4 {
			fmt.Println("Bye!")
			return
		} else {
			continue
		}
		serverResponse, _ = bufio.NewReader(connection).ReadString('\n')
		fmt.Print("Server Response: " + serverResponse)
	}
}

func sendFile(connection net.Conn, file *os.File) {
	sendBuffer := make([]byte, BUFFER_SIZE)
	for {
		_, err := file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		_, _ = connection.Write(sendBuffer)
	}
}

func receiveFile(connection net.Conn, fileName string, fileSize int64) bool {
	newFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("The file can't be created", err)
		return false
	}
	defer newFile.Close()
	var receivedBytes int64

	for {
		if (fileSize - receivedBytes) < BUFFER_SIZE {
			_, _ = io.CopyN(newFile, connection, fileSize-receivedBytes)
			_, _ = connection.Read(make([]byte, (receivedBytes+BUFFER_SIZE)-fileSize))
			break
		}
		_, _ = io.CopyN(newFile, connection, BUFFER_SIZE)
		receivedBytes += BUFFER_SIZE
	}
	return true
}
