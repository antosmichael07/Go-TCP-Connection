# Go-TCP-Connection

- A library meant to simplify the process of making TCP/UDP request and responses
- Install with `go get github.com/antosmichael07/Go-TCP-Connection`

## Rules

- Your events need to start from 4, events from 0 to 3 are reserved for client to server communication

## Example

### Server

```go
package main

import (
	"fmt"

	lgr "github.com/antosmichael07/Go-Logger"
	tcp "github.com/antosmichael07/Go-TCP-Connection"
)

const (
	event_test uint16 = iota + 4
)

func main() {
	server := tcp.NewServer("localhost:8080", "tcp")
	server.Logger.Level = lgr.Warning

	server.OnConnect(func(conn *tcp.Connection) {
		msg := []byte("Hello from the server")
		server.SendData(conn, event_test, &msg)
	})

	server.On(event_test, func(data *[]byte, conn *tcp.Connection) {
		fmt.Println("Received data from client:", string(*data))
	})

	server.Start()
}
```

### Client

```go
package main

import (
	"fmt"

	lgr "github.com/antosmichael07/Go-Logger"
	tcp "github.com/antosmichael07/Go-TCP-Connection"
)

const (
	event_test uint16 = iota + 4
)

func main() {
	client := tcp.NewClient("localhost:8080", "tcp")
	client.Logger.Level = lgr.Warning
	client.Connect()

	client.On(event_test, func(data *[]byte) {
		fmt.Println("Received data from the server: ", string(*data))

		msg := []byte("Hello from the client")
		client.SendData(event_test, &msg)
	})

	client.Listen()
}
```
