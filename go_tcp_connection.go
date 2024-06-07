package tcp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	Connections      []Connection
	Listener         net.Listener
	Address          string
	Logger           lgr.Logger
	Events           map[string]func([]byte, Connection)
	PossibleEvents   []string
	ShouldStop       bool
	OnConnectFunc    func(conn Connection)
	OnDisconnectFunc func(conn Connection)
	IsOnConnect      bool
	IsOnDisconnect   bool
}

type Connection struct {
	Connection net.Conn
	Token      string
}

type Client struct {
	Connection     net.Conn
	Address        string
	Logger         lgr.Logger
	Events         map[string]func([]byte)
	PossibleEvents []string
	ShouldStop     bool
	Token          string
	OnConnectFunc  func()
	IsOnConnect    bool
}

type Package struct {
	Token string
	Event string
	Data  []byte
}

func (pkg Package) ToByte() (bool, []byte) {
	logger := lgr.NewLogger("TCP")

	data, err := json.Marshal(pkg)
	if err != nil {
		logger.Log(lgr.Error, "Error marshaling package: %s", err)
		return false, []byte{}
	}

	return true, data
}

func NewServer(address string) Server {
	return Server{
		Connections:      []Connection{},
		Listener:         nil,
		Address:          address,
		Logger:           lgr.NewLogger("TCP"),
		Events:           make(map[string]func([]byte, Connection)),
		PossibleEvents:   []string{},
		ShouldStop:       false,
		OnConnectFunc:    func(conn Connection) {},
		OnDisconnectFunc: func(conn Connection) {},
		IsOnConnect:      false,
		IsOnDisconnect:   false,
	}
}

func NewClient(address string) Client {
	return Client{
		Connection:     nil,
		Address:        address,
		Logger:         lgr.NewLogger("TCP"),
		Events:         make(map[string]func([]byte)),
		PossibleEvents: []string{},
		ShouldStop:     false,
		Token:          "",
		OnConnectFunc:  func() {},
		IsOnConnect:    false,
	}
}

func (server *Server) Start() {
	listener, err := net.Listen("tcp", server.Address)
	if err != nil {
		panic(err)
	}
	server.Listener = listener
	server.Logger.Log(lgr.Info, "Server is listening on %s", server.Address)

	for !server.ShouldStop {
		conn, err := server.Listener.Accept()
		if err != nil {
			server.Logger.Log(lgr.Error, "Error accepting connection: %s", err)
		}

		go server.ReceiveData(conn)
	}
}

func (server *Server) Stop() {
	server.ShouldStop = true
	server.Listener.Close()
	server.Logger.Log(lgr.Info, "Server stopped")
}

func (server *Server) SendData(conn net.Conn, event string, data []byte) {
	can_send, to_send := Package{Event: event, Data: data}.ToByte()
	if !can_send {
		server.Logger.Log(lgr.Error, "Error creating package")
		return
	}
	_, err := conn.Write(to_send)
	if err != nil {
		server.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	server.Logger.Log(lgr.Info, "Data sent with the event name: %s", event)
}

func (server *Server) SendDataToAll(event string, data []byte) {
	for _, conn := range server.Connections {
		server.SendData(conn.Connection, event, data)
	}
}

func (server *Server) ReceiveData(conn net.Conn) {
	for !server.ShouldStop {
		data := make([]byte, 16384)
		n, err := conn.Read(data)
		data = data[:n]
		if err != nil {
			server.Logger.Log(lgr.Error, "Error reading data: %s", err)
			if server.IsOnDisconnect {
				for _, v := range server.Connections {
					if v.Connection == conn {
						server.OnDisconnectFunc(v)
						break
					}
				}
			}
			conn.Close()
			for i, v := range server.Connections {
				if v.Connection == conn {
					server.Connections = append(server.Connections[:i], server.Connections[i+1:]...)
					server.Logger.Log(lgr.Info, "Connection terminated")
					break
				}
			}
			return
		}

		pkg := Package{}
		err = json.Unmarshal(data, &pkg)
		if err != nil {
			server.Logger.Log(lgr.Error, "Error unmarshaling package: %s", err)
			server.SendData(conn, "error", []byte("Invalid data sent"))
			continue
		}

		is_token := false
		for _, v := range server.Connections {
			if v.Token == pkg.Token {
				is_token = true
				break
			}
		}
		if pkg.Event == "connect" && !is_token {
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

			server.Connections = append(server.Connections, Connection{Connection: conn, Token: token})
			server.Logger.Log(lgr.Info, "New connection: %s", token)
			server.OnConnectFunc(Connection{Connection: conn, Token: token})
			server.SendData(conn, "token", []byte(token))
			continue
		}
		if !is_token {
			server.Logger.Log(lgr.Warning, "Invalid token: %s", pkg.Token)
			server.SendData(conn, "error", []byte("Invalid token"))
			continue
		}

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

func (server *Server) On(event string, callback func([]byte, Connection)) {
	server.PossibleEvents = append(server.PossibleEvents, event)
	server.Events[event] = callback
}

func (server *Server) OnConnect(callback func(conn Connection)) {
	server.OnConnectFunc = callback
}

func (server *Server) OnDisconnect(callback func(conn Connection)) {
	server.OnDisconnectFunc = callback
}

func (client *Client) Connect() {
	conn, err := net.Dial("tcp", client.Address)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error connecting to server: %s", err)
	}
	client.Connection = conn
	client.Logger.Log(lgr.Info, "Connected to server")

	client.SendData("connect", []byte{})

	client.On("token", func(data []byte) {
		client.Logger.Log(lgr.Info, "Token received: %s", data)
		client.Token = string(data)
		go client.OnConnectFunc()
	})

	client.On("error", func(data []byte) {
		client.Logger.Log(lgr.Error, "Error received: %s", data)
	})
}

func (client *Client) Disconnect() {
	client.ShouldStop = true
	client.Connection.Close()
	client.Logger.Log(lgr.Info, "Connection closed")
}

func (client *Client) SendData(event string, data []byte) {
	can_send, to_send := Package{Token: client.Token, Event: event, Data: data}.ToByte()
	if !can_send {
		client.Logger.Log(lgr.Error, "Error creating package")
		return
	}
	_, err := client.Connection.Write(to_send)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data sent with the event name: %s", event)
}

func (client *Client) ReceiveData() {
	data := make([]byte, 16384)
	n, err := client.Connection.Read(data)
	data = data[:n]
	if err != nil {
		client.Logger.Log(lgr.Error, "Error reading data: %s", err)
		return
	}

	pkg := Package{}
	err = json.Unmarshal(data, &pkg)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error unmarshaling package: %s", err)
		return
	}

	client.Logger.Log(lgr.Info, "Data received with an event name: %s", pkg.Event)
	for _, event := range client.PossibleEvents {
		if event == pkg.Event {
			client.Events[pkg.Event](pkg.Data)
			break
		}
	}
}

func (client *Client) On(event string, callback func([]byte)) {
	client.PossibleEvents = append(client.PossibleEvents, event)
	client.Events[event] = callback
}

func (client *Client) Listen() {
	client.Logger.Log(lgr.Info, "Started listening")
	for !client.ShouldStop {
		client.ReceiveData()
	}
}

func (client *Client) OnConnect(callback func()) {
	client.OnConnectFunc = callback
}
