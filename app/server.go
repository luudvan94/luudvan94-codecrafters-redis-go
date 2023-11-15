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
	// var reader bufio.Reader

	for {
		// _, err := reader.ReadString('\n')
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }

		// var buf bytes.Buffer
		// wr := resp.NewWriter(&buf)
		// wr.WriteSimpleString("PONG")
		// fmt.Printf("%s\n", buf.String())
		conn.Write([]byte("+PONG\r\n"))
		return
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage

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
