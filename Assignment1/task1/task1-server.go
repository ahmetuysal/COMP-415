package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	BUFFER_SIZE                  = 65495
	CONNECTION_TYPE              = "tcp"
	CONNECTION_HOST              = "localhost"
	RESPONSE_LOGIN_SUCCESSFUL    = "Login successful."
	RESPONSE_INVALID_LOGIN       = "Invalid credentials."
	RESPONSE_MISSING_FILE        = "File does not exist."
	RESPONSE_STORE_SUCCESSFUL    = "%s stored successfully."
	RESPONSE_STORE_FAIL          = "%s store failed."
	RESPONSE_RETRIEVE_SUCCESSFUL = "%s found."
	RESPONSE_NOT_AUTHORIZED      = "You are not logged in."
	RESPONSE_FILE_READ_FAIL      = "File exists but cannot be reached at the moment."
)

func main() {
	port := os.Args[1]
	fmt.Println("port:" + port)

	listener, err := net.Listen(CONNECTION_TYPE, CONNECTION_HOST+":"+port)

	if err != nil {
		fmt.Println("Error when listening on address: "+CONNECTION_HOST+":"+port, err.Error())
		os.Exit(-1)
	}

	// close the listener when main returns
	defer listener.Close()

	// infinite loop for listening connections
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting client:", err.Error())
			continue
		}
		go HandleClientConnection(connection)
	}
}

func HandleClientConnection(connection net.Conn) {
	// close the connection when this function terminates
	defer connection.Close()
	var username string
	for {
		message, err := bufio.NewReader(connection).ReadString('\n')

		// client closed the connection
		// TODO: inspect error to validate connection is closed by client
		if err != nil {
			return
		}

		switch message[0] {
		// ASCII Codes 1-> 49, 2 -> 50, ...
		case 49:
			inputUsername := message[1 : len(message)-1]
			if len(inputUsername) > 0 && !strings.Contains(inputUsername, "\n") {
				username = inputUsername
				_, _ = connection.Write([]byte(RESPONSE_LOGIN_SUCCESSFUL + "\n"))
			} else {
				_, _ = connection.Write([]byte(RESPONSE_INVALID_LOGIN + "\n"))
			}
		case 50:
			separatorIndex := strings.Index(message, ":")
			fileSize, _ := strconv.ParseInt(message[1:separatorIndex], 10, 64)
			fileName := message[separatorIndex+1 : len(message)-1]
			fileReceiveSuccess := receiveFile(connection, username, fileName, fileSize)
			if fileReceiveSuccess {
				_, _ = connection.Write([]byte(fmt.Sprintf(RESPONSE_STORE_SUCCESSFUL, fileName) + "\n"))
			} else {
				_, _ = connection.Write([]byte(fmt.Sprintf(RESPONSE_STORE_FAIL, fileName) + "\n"))
			}
		case 51:
			if len(username) == 0 {
				_, _ = connection.Write([]byte(RESPONSE_NOT_AUTHORIZED + "\n"))
				continue
			}

			fileName := message[1 : len(message)-1]
			fmt.Println(fileName)
			file, err := os.Open(username + "/" + fileName)
			if err != nil {
				_, _ = connection.Write([]byte(RESPONSE_MISSING_FILE + "\n"))
				continue
			}

			fileInfo, err := file.Stat()
			if err != nil {
				_ = file.Close()
				_, _ = connection.Write([]byte(RESPONSE_FILE_READ_FAIL + "\n"))
				continue
			}

			fileSize := strconv.FormatInt(fileInfo.Size(), 10)
			fileName = fileInfo.Name()
			_, _ = fmt.Fprintf(connection, "3"+fileSize+":"+fileName+"\n")
			sendFile(connection, file)
			_ = file.Close()
			_, _ = connection.Write([]byte(fmt.Sprintf(RESPONSE_RETRIEVE_SUCCESSFUL, fileName) + "\n"))
		case 52:
		}
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

func receiveFile(connection net.Conn, folder string, fileName string, fileSize int64) bool {
	CreateDirIfNotExist(folder)

	newFile, err := os.Create(folder + "/" + fileName)
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

// Work of Siong-Ui Te: https://siongui.github.io/2017/03/28/go-create-directory-if-not-exist/
func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}
