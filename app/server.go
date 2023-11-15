package main

import (
	"fmt"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func handleConnection(conn net.Conn) {
	fmt.Println("here")
	defer conn.Close()

	for {
		conn.Write([]byte("+PONG\r\n"))
		return
	}
}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}

}
