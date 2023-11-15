package main

import (
	"bytes"
	"fmt"
	"io"

	// Uncomment this block to pass the first stage
	"net"
	"os"

	"github.com/tidwall/resp"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	rd := resp.NewReader(conn)
	var buf bytes.Buffer
	wr := resp.NewWriter(&buf)

	for {
		v, _, err := rd.ReadValue()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading data: ", err)
			os.Exit(1)
		}
		fmt.Printf("Read %s\n", v.Type())
		if v.Type() == resp.SimpleString {
			switch v.String() {
			case "PING":
				wr.WriteSimpleString("PONG")
				conn.Write(buf.Bytes())
			default:
				fmt.Printf("Unknown command: %s\n", v)
			}
		}
		if v.Type() == resp.Array && len(v.Array()) >= 2 {
			command := v.Array()[0]
			value := v.Array()[1]
			switch command.String() {
			case "ECHO":
				conn.Write(value.Bytes())
			}
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
