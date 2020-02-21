package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const (
	BUFFER_SIZE = 65495
	CONNECTION_TYPE = "tcp"
	CONNECTION_HOST = "localhost"
)
func main() {
	port := os.Args[1]
	fmt.Println("port:" + port)

	listener, err := net.Listen(CONNECTION_TYPE, CONNECTION_HOST + ":" + port)

	if err != nil {
		fmt.Println("Error when listening on address: " + CONNECTION_HOST + ":" + port, err.Error())
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

func HandleClientConnection(connection net.Conn)  {
	// close the connection when this function terminates
	defer connection.Close()
	var username string
	for {
		message, _ := bufio.NewReader(connection).ReadString('\n')

		switch message[0] {
		// ASCII Codes 1-> 49, 2 -> 50, ...
		case 49:
			username = message[1:len(message) - 1]
		case 50:

		case 51:

		case 52:
		}
		fmt.Println(len(username))
	}

}
