package tcp

import (
	"math/rand"
	"net"
	"strings"
	"time"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	// The protocol to use
	Protocol string
	// Here are the connections and the tokens of the clients
	Connections []Connection
	// Listener of the server
	Listener net.Listener
	// Address of the server
	Address string
	// Logger is the logger of the server, can be customized (https://github.com/antosmichael07/Go-Logger)
	Logger lgr.Logger
	// Here are the events that the server can handle
	Events         [65536]func(*[]byte, *Connection)
	PossibleEvents []uint16
	// ShouldStop is a boolean that is used to stop the server
	ShouldStop bool
	// OnConnectFunc is the function that is called when a client connects if IsOnConnect is true
	// Should be initialized with the function OnConnect to automatically set the IsOnConnect to true
	OnConnectFunc func(conn *Connection)
	IsOnConnect   bool
	// OnDisconnectFunc is the function that is called when a client disconnects if IsOnDisconnect is true
	// Should be initialized with the function OnDisconnect to automatically set the IsOnDisconnect to true
	OnDisconnectFunc func(conn *Connection)
	IsOnDisconnect   bool
	// OnStartFunc is the function that is called when the server starts if IsOnStart is true
	// Should be initialized with the function OnStart to automatically set the IsOnStart to true
	OnStartFunc func()
	IsOnStart   bool

	ShouldListen bool
}

// Connection is a struct that contains the connection and the token of the client
type Connection struct {
	Connection   net.Conn
	Token        [64]byte
	ReceivedLast bool
	Queue        [][]byte
	ShouldClose  bool
	IsOK         bool
}

// toByte is a function that converts the package to a byte array to be sent
func (pkg Package) toByteServer() (data []byte) {
	data = make([]byte, 74+len(pkg.Data))
	size := uint64(74 + len(pkg.Data))

	for i := 0; i < 8; i++ {
		data[i] = byte(size >> (8 * i))
	}
	for i := 0; i < 2; i++ {
		data[i+72] = byte(pkg.Event >> (8 * i))
	}
	for i := 0; i < len(pkg.Data); i++ {
		data[i+74] = pkg.Data[i]
	}

	return data
}

// NewServer is a function that creates a new server with the given address
func NewServer(address, protocol string) Server {
	logger, _ := lgr.NewLogger(strings.ToUpper(protocol), "", false)

	return Server{
		Protocol:         protocol,
		Connections:      []Connection{},
		Listener:         nil,
		Address:          address,
		Logger:           logger,
		Events:           [65536]func(*[]byte, *Connection){},
		PossibleEvents:   []uint16{},
		ShouldStop:       false,
		OnConnectFunc:    func(conn *Connection) {},
		IsOnConnect:      false,
		OnDisconnectFunc: func(conn *Connection) {},
		IsOnDisconnect:   false,
		OnStartFunc:      func() {},
		IsOnStart:        false,
		ShouldListen:     true,
	}
}

// fromByte is a function that converts the byte array to a package
func (pkg *Package) fromByte(data *[]byte) {
	pkg.Size = 0
	for i := 0; i < 8; i++ {
		pkg.Size |= uint64((*data)[i]) << (8 * i)
	}
	for i := 0; i < 64; i++ {
		pkg.Token[i] = (*data)[i+8]
	}
	pkg.Event = 0
	for i := 0; i < 2; i++ {
		pkg.Event |= uint16((*data)[i+72]) << (8 * i)
	}
	pkg.Data = (*data)[74:]
}

// Start is a function that starts the server, listens for connections and receives data
// Should be called after you have set the events
func (server *Server) Start() error {
	// Start listening
	listener, err := net.Listen(server.Protocol, server.Address)
	if err != nil {
		return err
	}
	// Save the listener
	server.Listener = listener
	server.Logger.Log(lgr.Info, "Server is listening on %s", server.Address)

	// Event on data receivment
	server.On(event_last_data_received, func(data *[]byte, conn *Connection) {
		conn.ReceivedLast = true
	})

	server.On(event_are_you_ok, func(data *[]byte, conn *Connection) {
		for i := range server.Connections {
			if server.Connections[i].Token == conn.Token {
				server.Connections[i].IsOK = true
				break
			}
		}
	})

	go func() {
		for !server.ShouldStop {
			for i := 0; i < len(server.Connections); i++ {
				if !server.Connections[i].ShouldClose {
					if !server.Connections[i].IsOK {
						server.Connections[i].ShouldClose = true
					} else {
						server.Connections[i].IsOK = false
						server.SendData(&server.Connections[i], event_are_you_ok, &[]byte{})
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Start receiving data
	go func() {
		for !server.ShouldStop {
			for !server.ShouldListen {
				time.Sleep(1 * time.Second)
				if server.ShouldStop {
					break
				}
			}
			conn, err := server.Listener.Accept()
			if server.ShouldStop {
				break
			}
			if !server.ShouldListen {
				conn.Close()
				continue
			}

			if err != nil {
				server.Logger.Log(lgr.Error, "Error accepting connection: %s", err)
			}

			go server.ReceiveData(conn)
		}
	}()

	// Call the OnStart function
	if server.IsOnStart {
		server.OnStartFunc()
	}

	// Start sending data
	for !server.ShouldStop {
		for i := 0; i < len(server.Connections); i++ {
			if server.Connections[i].ShouldClose {
				server.Connections[i].Connection.Close()
				server.OnDisconnectFunc(&server.Connections[i])
				server.Connections = append(server.Connections[:i], server.Connections[i+1:]...)
				server.Logger.Log(lgr.Info, "Connection terminated")
				i--
				continue
			}
			if server.Connections[i].Queue != nil && (len(server.Connections[i].Queue) != 0 && server.Connections[i].ReceivedLast || len(server.Connections[i].Queue) > 5) {
				server.actuallySendData(&server.Connections[i], &server.Connections[i].Queue[0])
				server.Connections[i].Queue = server.Connections[i].Queue[1:]
			}
		}
	}

	return nil
}

// Stop is a function that stops the server
func (server *Server) Stop() {
	server.ShouldStop = true
	server.Listener.Close()
	server.Logger.Log(lgr.Info, "Server stopped")
}

// ReceiveData is a function that receives data from a specific connection
func (server *Server) ReceiveData(conn net.Conn) {
	for !server.ShouldStop {
		// Read the data
		data := make([]byte, 16384)
		n, err := conn.Read(data)
		data = data[:n]
		// If there is an error, close the connection, remove it from the connections and call the OnDisconnect function
		if err != nil {
			server.Logger.Log(lgr.Error, "Error reading data: %s", err)
			// Call the OnDisconnect function
			if server.IsOnDisconnect {
				for i := range server.Connections {
					if server.Connections[i].Connection == conn {
						server.OnDisconnectFunc(&server.Connections[i])
						break
					}
				}
			}
			// Remove the connection from the connections list
			for i := range server.Connections {
				if server.Connections[i].Connection == conn {
					server.Connections[i].ShouldClose = true
					break
				}
			}
			// Close the connection
			conn.Close()
			return
		}

		// Decode the data
		if len(data) < 74 {
			server.Logger.Log(lgr.Error, "Invalid data received: %v", data)
			continue
		}
		pkg := Package{}
		pkg.fromByte(&data)

		if pkg.Size > uint64(len(data)) {
			server.Logger.Log(lgr.Error, "Invalid data received: %v", data)
			continue
		}
		pkg.Data = pkg.Data[:pkg.Size-74]

		// Check if the token is valid
		is_token := false
		for i := range server.Connections {
			if server.Connections[i].Token == pkg.Token {
				is_token = true
				break
			}
		}
		// If the event is connect, create a token and add the connection to the connections list
		if pkg.Event == event_connect && !is_token {
			// Create a token
			token := [64]byte{}
			for token == [64]byte{} {
				for i := 0; i < 64; i++ {
					token[i] = byte(rand.Intn(255))
				}
				for _, v := range server.Connections {
					if v.Token == token {
						token = [64]byte{}
						break
					}
				}
			}

			// Add the connection to the connections list
			server.Connections = append(server.Connections, Connection{Connection: conn, Token: token, ReceivedLast: true, Queue: [][]byte{}, ShouldClose: false, IsOK: true})
			server.Logger.Log(lgr.Info, "New connection: %v", token)
			// Send the token to the client
			token_slice := []byte(token[:])
			server.SendData(&server.Connections[len(server.Connections)-1], event_token, &token_slice)
			// Call the OnConnect function
			if server.IsOnConnect {
				server.OnConnectFunc(&server.Connections[len(server.Connections)-1])
			}
			continue
		}
		// If the token is invalid, send an error
		if !is_token {
			server.Logger.Log(lgr.Warning, "Invalid token: %v", pkg.Token)
			continue
		}

		// If the event is valid, call the function that is associated with the event
		server.Logger.Log(lgr.Info, "Data received with an event name: %v", pkg.Event)
		for i := range server.PossibleEvents {
			if server.PossibleEvents[i] == pkg.Event {
				for i := range server.Connections {
					if server.Connections[i].Token == pkg.Token {
						server.Events[pkg.Event](&pkg.Data, &server.Connections[i])
						break
					}
				}
			}
		}
	}
}

// SendData to queue data to be sent to a specific connection with the given event name, and data
func (server *Server) SendData(conn *Connection, event uint16, data *[]byte) {
	conn.Queue = append(conn.Queue, Package{Event: event, Data: *data}.toByteServer())
}

// SendDataToAll is a function that sends data to all the connections, with the given event name, and data
func (server *Server) SendDataToAll(event uint16, data *[]byte) {
	for i := range server.Connections {
		server.SendData(&server.Connections[i], event, data)
	}
}

// actuallySendData is a function that sends data to a specific connectionl, with the given event name, and data
func (server *Server) actuallySendData(conn *Connection, data *[]byte) {
	// Send the data
	_, err := conn.Connection.Write(*data)
	if err != nil || len(*data) < 74 {
		server.Logger.Log(lgr.Error, "Error sending data: %s", err)
	} else {
		server.Logger.Log(lgr.Info, "Data sent with the event name: %v", uint16((*data)[72])|(uint16((*data)[73])<<8))
	}
	// Set the ReceivedLast to false
	conn.ReceivedLast = false
}

// On is a function that adds an event to the server
func (server *Server) On(event uint16, callback func(*[]byte, *Connection)) {
	server.PossibleEvents = append(server.PossibleEvents, event)
	server.Events[event] = callback
}

// OnStart is a function that sets the OnStartFunc and IsOnStart to true
func (server *Server) OnStart(callback func()) {
	server.OnStartFunc = callback
	server.IsOnStart = true
}

// OnConnect is a function that sets the OnConnectFunc and IsOnConnect to true
func (server *Server) OnConnect(callback func(conn *Connection)) {
	server.OnConnectFunc = callback
	server.IsOnConnect = true
}

// OnDisconnect is a function that sets the OnDisconnectFunc and IsOnDisconnect to true
func (server *Server) OnDisconnect(callback func(conn *Connection)) {
	server.OnDisconnectFunc = callback
	server.IsOnDisconnect = true
}
