package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	// Uncomment this block to pass the first stage
	"net"
	"os"

	"github.com/hdt3213/rdb/parser"
	"github.com/tidwall/resp"
)

type Connection struct {
	*resp.Reader
	*resp.Writer
	base net.Conn
}

type Server struct {
	mu       sync.RWMutex
	kvs      map[string]Value
	dbConfig map[string]string
}

type Value struct {
	value  resp.Value
	expire *time.Time
}

func NewServer(dbConfig map[string]string) *Server {
	server := &Server{kvs: make(map[string]Value)}
	server.dbConfig = dbConfig
	server.parseDB()
	return server
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

func (server *Server) get(key string) (Value, bool) {
	server.mu.RLock()
	value, ok := server.kvs[key]
	server.mu.RUnlock()
	fmt.Printf("Key: %s Value: %s\n", key, value.value.String())

	now := time.Now()
	if ok && (!value.expire.IsZero() && value.expire.Before(now)) {
		fmt.Printf("Expired: %s\n", value.expire.String())
		delete(server.kvs, key)
		return value, false
	}

	return value, ok
}

func (server *Server) parseDB() {
	dir, exist := server.dbConfig["dir"]
	if !exist {
		fmt.Println("Can not found db directory")
		dir = "./"
	}

	fileName, exist := server.dbConfig["dbfilename"]
	if !exist {
		fmt.Println("Can not found db file name")
		fileName = "dumb.rdb"
	}

	filePath := strings.Join([]string{dir, fileName}, "/")

	rdbFile, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Can not found %s\n", filePath)
		return
	}
	defer func() {
		_ = rdbFile.Close()
	}()
	decoder := parser.NewDecoder(rdbFile)
	err = decoder.Parse(func(o parser.RedisObject) bool {
		switch o.GetType() {
		case parser.StringType:
			str := o.(*parser.StringObject)
			server.kvs[str.Key] = Value{value: resp.StringValue(string(str.Value)), expire: str.Expiration}
		}
		// return true to continue, return false to stop the iteration
		return true
	})
	if err != nil {
		panic(err)
	}

}

func (server *Server) set(args []resp.Value) {
	newValue := Value{}
	newValue.value = args[2]

	if len(args) >= 5 && args[3].String() == "px" {
		expiryAmount := args[4].Integer()
		newTime := time.Now().Add(time.Duration(expiryAmount) * time.Millisecond)
		newValue.expire = &newTime
		fmt.Printf("Expiration time: %s\n", newValue.expire.String())
	}
	server.mu.Lock()
	server.kvs[args[1].String()] = newValue
	server.mu.Unlock()
	fmt.Printf("set %s with %s\n", args[1].String(), args[2].String())
}

func (server *Server) HandleConnection(conn *Connection) {
	defer conn.Close()

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
				if len(values) < 3 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'set' command"))
				} else {
					server.set(values)
					conn.WriteSimpleString("OK")
				}
				continue
			case "get":
				if len(values) != 2 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'get' command"))
				} else {
					s, ok := server.get(values[1].String())
					if !ok {
						conn.WriteNull()
					} else {
						conn.WriteString(s.value.String())
					}
				}
				continue
			case "config":
				if len(values) < 3 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'CONFIG' command"))
				} else {
					switch strings.ToLower(values[1].String()) {
					case "get":
						key := values[2]
						value, exist := server.dbConfig[key.String()]
						if !exist {
							conn.WriteError(fmt.Errorf("ERR can not found %s", values[2]))
						} else {
							conn.WriteArray([]resp.Value{key, resp.StringValue(value)})
						}
						continue
					}
				}
				continue
			case "keys":
				if len(values) < 2 {
					conn.WriteError(errors.New("ERR wrong number of arguments for 'keys' command"))
				} else {
					if values[1].String() == "*" {
						var keys []resp.Value
						for key := range server.kvs {
							keys = append(keys, resp.StringValue(key))
						}

						conn.WriteArray(keys)
					} else {
						value, exist := server.kvs[values[1].String()]
						if !exist {
							conn.WriteError(fmt.Errorf("can not found key %s", values[1].String()))
						} else {
							conn.WriteString(value.value.String())
						}

					}

				}
				continue
			}

		}
	}
}

func GetArgumentName(original string) string {
	prefix := "--"
	if strings.HasPrefix(original, prefix) {
		return original[len(prefix):]
	}
	return original
}

func main() {
	dbConfig := map[string]string{}

	for i, arg := range os.Args {
		if arg == "--dir" {
			fmt.Printf("--dir: %s\n", os.Args[i+1])
			dbConfig[GetArgumentName(arg)] = os.Args[i+1]
		} else if arg == "--dbfilename" {
			fmt.Printf("--dbfilename: %s\n", os.Args[i+1])
			dbConfig[GetArgumentName(arg)] = os.Args[i+1]
		}
	}

	server := NewServer(dbConfig)
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
