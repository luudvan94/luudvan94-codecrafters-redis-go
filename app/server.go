package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	// Uncomment this block to pass the first stage
	"net"
	"os"

	"github.com/tidwall/resp"
)

type Connection struct {
	*resp.Reader
	*resp.Writer
	base net.Conn
}

type Server struct {
	mu  sync.RWMutex
	kvs map[string]resp.Value
}

func NewServer() *Server {
	return &Server{kvs: make(map[string]resp.Value)}
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		Reader: resp.NewReader(conn),
		Writer: resp.NewWriter(conn),
		base:   conn,
	}
}

func (conn *Connection) Close() {
	conn.base.Close()
}

func (server *Server) HandleConnection(conn *Connection) {
	defer conn.Close()

	// rd := resp.NewReader(conn)
	// var buf bytes.Buffer
	// wr := resp.NewWriter(&buf)

	for {
		v, _, _, err := conn.ReadMultiBulk()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading data: ", err)
			os.Exit(1)
		}

		fmt.Printf("Read %s %d\n", v.Type(), len(v.Array()))
		if v.Type() == resp.Array && len(v.Array()) > 0 {
			values := v.Array()
			command := values[0]
			switch strings.ToLower(command.String()) {
			case "echo":
				if err := conn.WriteString(values[1].String()); err != nil {
					fmt.Println(err)
				}
				continue
			case "ping":
				conn.WriteSimpleString("PONG")
				continue
			case "set":
				if len(values) != 3 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'set' command"))
				} else {
					server.mu.Lock()
					server.kvs[values[1].String()] = values[2]
					server.mu.Unlock()
					conn.WriteSimpleString("OK")
				}
				continue
			case "get":
				if len(values) != 2 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'get' command"))
				} else {
					server.mu.RLock()
					s, ok := server.kvs[values[1].String()]
					server.mu.RUnlock()
					if !ok {
						conn.WriteNull()
					} else {
						conn.WriteString(s.String())
					}
				}
				continue
			}
		}
	}
}

func main() {
	server := NewServer()
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

		go server.HandleConnection(NewConnection(conn))
	}

}
