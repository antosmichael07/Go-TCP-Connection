package tcp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	// Here are the connections and the tokens of the clients
	Connections []Connection
	// Listener of the server
	Listener net.Listener
	// Address of the server
	Address string
	// Logger is the logger of the server, can be customized (https://github.com/antosmichael07/Go-Logger)
	Logger lgr.Logger
	// Here are the events that the server can handle
	Events         map[string]func([]byte, Connection)
	PossibleEvents []string
	// ShouldStop is a boolean that is used to stop the server
	ShouldStop bool
	// OnConnectFunc is the function that is called when a client connects if IsOnConnect is true
	// Should be initialized with the function OnConnect to automatically set the IsOnConnect to true
	OnConnectFunc func(conn Connection)
	IsOnConnect   bool
	// OnDisconnectFunc is the function that is called when a client disconnects if IsOnDisconnect is true
	// Should be initialized with the function OnDisconnect to automatically set the IsOnDisconnect to true
	OnDisconnectFunc func(conn Connection)
	IsOnDisconnect   bool
}

// Connection is a struct that contains the connection and the token of the client
type Connection struct {
	Connection net.Conn
	Token      string
}

type Client struct {
	// Connection to the server
	Connection net.Conn
	// Address of the server
	Address string
	// Logger is the logger of the client, can be customized (https://github.com/antosmichael07/Go-Logger)
	Logger lgr.Logger
	// Here are the events that the client can handle
	Events         map[string]func([]byte)
	PossibleEvents []string
	// ShouldStop is a boolean that is used to stop the client
	ShouldStop bool
	// Token is the token of the client
	Token string
	// OnConnectFunc is the function that is called when the client connects if IsOnConnect is true
	OnConnectFunc func()
	IsOnConnect   bool
}

// Package is a struct that contains the token, the event and the data that is sent
type Package struct {
	Token string
	Event string
	Data  []byte
}

// ToByte is a function that converts the package to a byte array to be sent
func (pkg Package) ToByte(logger lgr.Logger) (bool, []byte) {
	data, err := json.Marshal(pkg)
	if err != nil {
		logger.Log(lgr.Error, "Error marshaling package: %s", err)
		return false, []byte{}
	}

	return true, data
}

// NewServer is a function that creates a new server with the given address
func NewServer(address string) Server {
	logger := lgr.NewLogger("TCP")
	logger.Output.File = false

	return Server{
		Connections:      []Connection{},
		Listener:         nil,
		Address:          address,
		Logger:           logger,
		Events:           map[string]func([]byte, Connection){},
		PossibleEvents:   []string{},
		ShouldStop:       false,
		OnConnectFunc:    func(conn Connection) {},
		IsOnConnect:      false,
		OnDisconnectFunc: func(conn Connection) {},
		IsOnDisconnect:   false,
	}
}

// NewClient is a function that creates a new client with the given address
func NewClient(address string) Client {
	logger := lgr.NewLogger("TCP")
	logger.Output.File = false

	return Client{
		Connection:     nil,
		Address:        address,
		Logger:         logger,
		Events:         map[string]func([]byte){},
		PossibleEvents: []string{},
		ShouldStop:     false,
		Token:          "",
		OnConnectFunc:  func() {},
		IsOnConnect:    false,
	}
}

// Start is a function that starts the server, listens for connections and receives data
// Should be called after you have set the events
func (server *Server) Start() {
	// Start listening
	listener, err := net.Listen("tcp", server.Address)
	if err != nil {
		panic(err)
	}
	// Save the listener
	server.Listener = listener
	server.Logger.Log(lgr.Info, "Server is listening on %s", server.Address)

	// Start receiving data
	for !server.ShouldStop {
		conn, err := server.Listener.Accept()
		if err != nil {
			server.Logger.Log(lgr.Error, "Error accepting connection: %s", err)
		}
		if server.ShouldStop {
			break
		}

		go server.ReceiveData(conn)
	}
}

// Stop is a function that stops the server
func (server *Server) Stop() {
	server.ShouldStop = true
	server.Listener.Close()
	server.Logger.Log(lgr.Info, "Server stopped")
}

// SendData is a function that sends data to a specific connectionl, with the given event name, and data
func (server *Server) SendData(conn net.Conn, event string, data []byte) {
	// Convert the package to a byte array
	can_send, to_send := Package{Event: event, Data: data}.ToByte(server.Logger)
	if !can_send {
		server.Logger.Log(lgr.Error, "Error creating package")
		return
	}
	// Send the data
	_, err := conn.Write(to_send)
	if err != nil {
		server.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	server.Logger.Log(lgr.Info, "Data sent with the event name: %s", event)
}

// SendDataToAll is a function that sends data to all the connections, with the given event name, and data
func (server *Server) SendDataToAll(event string, data []byte) {
	for _, conn := range server.Connections {
		server.SendData(conn.Connection, event, data)
	}
}

// ReceiveData is a function that receives data from a specific connection
func (server *Server) ReceiveData(conn net.Conn) {
	for !server.ShouldStop {
		// Read the data
		data := make([]byte, 16384)
		n, err := conn.Read(data)
		if server.ShouldStop {
			break
		}
		data = data[:n]
		// If there is an error, close the connection, remove it from the connections and call the OnDisconnect function
		if err != nil {
			server.Logger.Log(lgr.Error, "Error reading data: %s", err)
			// Call the OnDisconnect function
			if server.IsOnDisconnect {
				for _, v := range server.Connections {
					if v.Connection == conn {
						server.OnDisconnectFunc(v)
						break
					}
				}
			}
			// Remove the connection from the connections list
			for i, v := range server.Connections {
				if v.Connection == conn {
					server.Connections = append(server.Connections[:i], server.Connections[i+1:]...)
					server.Logger.Log(lgr.Info, "Connection terminated")
					break
				}
			}
			// Close the connection
			conn.Close()
			return
		}

		// Unmarshal the data
		pkg := Package{}
		err = json.Unmarshal(data, &pkg)
		if err != nil {
			server.Logger.Log(lgr.Error, "Error unmarshaling package: %s", err)
			server.SendData(conn, "error", []byte("Invalid data sent"))
			continue
		}

		// Check if the token is valid
		is_token := false
		for _, v := range server.Connections {
			if v.Token == pkg.Token {
				is_token = true
				break
			}
		}
		// If the event is connect, create a token and add the connection to the connections list
		if pkg.Event == "connect" && !is_token {
			// Create a token
			token := ""
			for token == "" {
				for i := 0; i < 32; i++ {
					token = fmt.Sprintf("%s%d", token, rand.Intn(9))
				}
				for _, v := range server.Connections {
					if v.Token == token {
						token = ""
						break
					}
				}
			}

			// Add the connection to the connections list
			server.Connections = append(server.Connections, Connection{Connection: conn, Token: token})
			server.Logger.Log(lgr.Info, "New connection: %s", token)
			// Send the token to the client
			server.SendData(conn, "token", []byte(token))
			// Call the OnConnect function
			if server.IsOnConnect {
				server.OnConnectFunc(Connection{Connection: conn, Token: token})
			}
			continue
		}
		// If the token is invalid, send an error
		if !is_token {
			server.Logger.Log(lgr.Warning, "Invalid token: %s", pkg.Token)
			server.SendData(conn, "error", []byte("Invalid token"))
			continue
		}

		// If the event is valid, call the function that is associated with the event
		server.Logger.Log(lgr.Info, "Data received with an event name: %s", pkg.Event)
		for _, event := range server.PossibleEvents {
			if event == pkg.Event {
				for _, v := range server.Connections {
					if v.Token == pkg.Token {
						server.Events[pkg.Event](pkg.Data, v)
						break
					}
				}
			}
		}
	}
}

// On is a function that adds an event to the server
func (server *Server) On(event string, callback func([]byte, Connection)) {
	server.PossibleEvents = append(server.PossibleEvents, event)
	server.Events[event] = callback
}

// OnConnect is a function that sets the OnConnectFunc and IsOnConnect to true
func (server *Server) OnConnect(callback func(conn Connection)) {
	server.OnConnectFunc = callback
	server.IsOnConnect = true
}

// OnDisconnect is a function that sets the OnDisconnectFunc and IsOnDisconnect to true
func (server *Server) OnDisconnect(callback func(conn Connection)) {
	server.OnDisconnectFunc = callback
	server.IsOnDisconnect = true
}

// Connect is a function that connects the client to the server
func (client *Client) Connect() {
	// Connect to the server
	conn, err := net.Dial("tcp", client.Address)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error connecting to server: %s", err)
	}
	// Save the connection
	client.Connection = conn
	client.Logger.Log(lgr.Info, "Connected to server")

	// Send the connect event
	client.SendData("connect", []byte{})

	// Receive the token
	client.On("token", func(data []byte) {
		client.Logger.Log(lgr.Info, "Token received: %s", data)
		// Save the token
		client.Token = string(data)
		// Call the OnConnect function
		if client.IsOnConnect {
			go client.OnConnectFunc()
		}
	})

	// Receive the error event
	client.On("error", func(data []byte) {
		client.Logger.Log(lgr.Error, "Error received: %s", data)
	})
}

// Disconnect is a function that disconnects the client from the server
func (client *Client) Disconnect() {
	client.ShouldStop = true
	client.Connection.Close()
	client.Logger.Log(lgr.Info, "Connection closed")
}

// SendData is a function that sends data to the server with the given event name, and data
func (client *Client) SendData(event string, data []byte) {
	// Convert the package to a byte array
	can_send, to_send := Package{Token: client.Token, Event: event, Data: data}.ToByte(client.Logger)
	if !can_send {
		client.Logger.Log(lgr.Error, "Error creating package")
		return
	}
	// Send the data
	_, err := client.Connection.Write(to_send)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data sent with the event name: %s", event)
}

// ReceiveData is a function that receives data from the server
func (client *Client) ReceiveData() {
	// Read the data
	data := make([]byte, 16384)
	n, err := client.Connection.Read(data)
	if client.ShouldStop {
		return
	}
	data = data[:n]
	if err != nil {
		client.Logger.Log(lgr.Error, "Error reading data: %s", err)
		return
	}

	// Unmarshal the data
	pkg := Package{}
	err = json.Unmarshal(data, &pkg)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error unmarshaling package: %s", err)
		return
	}

	// If the event is valid, call the function that is associated with the event
	client.Logger.Log(lgr.Info, "Data received with an event name: %s", pkg.Event)
	for _, event := range client.PossibleEvents {
		if event == pkg.Event {
			client.Events[pkg.Event](pkg.Data)
			break
		}
	}
}

// On is a function that adds an event to the client
func (client *Client) On(event string, callback func([]byte)) {
	client.PossibleEvents = append(client.PossibleEvents, event)
	client.Events[event] = callback
}

// Listen is a function that listens for data from the server
// Should be called after you have set the events
func (client *Client) Listen() {
	client.Logger.Log(lgr.Info, "Started listening")
	for !client.ShouldStop {
		client.ReceiveData()
	}
}

// OnConnect is a function that sets the OnConnectFunc and IsOnConnect to true
func (client *Client) OnConnect(callback func()) {
	client.OnConnectFunc = callback
	client.IsOnConnect = true
}
