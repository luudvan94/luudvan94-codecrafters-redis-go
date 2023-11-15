package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection failed: ", err)
				break
			}
			fmt.Println("Error reading data")
			os.Exit(1)
		}

		fmt.Println("Input: ", input)
		if strings.EqualFold(input, "PING\r\n") {
			response := "+PONG\r\n"
			_, err = conn.Write([]byte(response))
			if err != nil {
				fmt.Println("Error writing data: ", err)
				os.Exit(1)
			}
			fmt.Println("Response: ", response)
		} else {
			fmt.Println("Unknown command: ", input)
		}
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
