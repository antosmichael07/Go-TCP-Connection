# Go-TCP-Connection

A library meant to simplify the process of making TCP request and responses.<hr>
Install with `go get github.com/antosmichael07/Go-TCP-Connection`

## Example

### Server

```go
package main

import (
	"fmt"

	tcp "github.com/antosmichael07/Go-TCP-Connection"
)

func main() {
	// Create a new server
	server := tcp.NewServer("localhost:8080")

	// Create some events
	// This event will be triggered when a client connects to the server
	server.OnConnect(func(conn tcp.Connection) {
		// Send data to the client that just connected
		server.SendData(conn.Connection, "test", []byte("Hello from server"))
	})

	// This event will be triggered when a client disconnects from the server
	server.OnDisconnect(func(conn tcp.Connection) {
		fmt.Println("Client disconnected")
	})

	// This event will be triggered when the server receives data from a client with the event name "test"
	server.On("test", func(data []byte, conn tcp.Connection) {
		fmt.Println("Received data from client:", string(data))

		// Send data back to the client
		server.SendData(conn.Connection, "test", []byte("Hello from server"))

		// Send data to all clients
		server.SendDataToAll("test", []byte("Hello from server"))
	})

	// This event will be triggered when the server receives data from a client with the event name "stop"
	server.On("stop", func(data []byte, conn tcp.Connection) {
		// Stops the server
		server.Stop()
	})

	// Start the server
	server.Start()
}
```

### Client

```go
package main

import (
	"fmt"

	tcp "github.com/antosmichael07/Go-TCP-Connection"
)

func main() {
	// Create a client
	client := tcp.NewClient("localhost:8080")

	// Connect to the server
	client.Connect()

	// Receive data from the server with the event name "test"
	client.On("test", func(data []byte) {
		fmt.Println("Received data from the server: ", string(data))

		// Send data to the server with the event name "test"
		client.SendData("test", []byte("Hello from the client"))
	})

	// Receive data from the server with the event name "stop"
	client.On("stop", func(data []byte) {
		// Disconnect from the server
		client.Disconnect()
	})

	// When the client connects to the server
	client.OnConnect(func() {
		fmt.Println("Connected to the server")
	})

	// Start listening for incoming data
	client.Listen()
}
```
