package tcp

import (
	"net"
	"strings"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	Connection     net.Conn
	Listener       net.Listener
	Address        string
	Logger         lgr.Logger
	Events         map[string]func(string)
	PossibleEvents []string
	ShuoldStop     bool
}

type Client struct {
	Connection     net.Conn
	Address        string
	Logger         lgr.Logger
	Events         map[string]func(string)
	PossibleEvents []string
	ShouldStop     bool
}

type Package struct {
	Event string
	Data  string
}

func (pkg Package) String() string {
	return pkg.Event + "|" + string(pkg.Data)
}

func NewServer(address string) Server {
	return Server{
		Connection:     nil,
		Listener:       nil,
		Address:        address,
		Logger:         lgr.NewLogger("TCP"),
		Events:         make(map[string]func(string)),
		PossibleEvents: []string{},
		ShuoldStop:     false,
	}
}

func NewClient(address string) Client {
	return Client{
		Connection:     nil,
		Address:        address,
		Logger:         lgr.NewLogger("TCP"),
		Events:         make(map[string]func(string)),
		PossibleEvents: []string{},
		ShouldStop:     false,
	}
}

func (server *Server) Start() {
	listener, err := net.Listen("tcp", server.Address)
	if err != nil {
		panic(err)
	}
	server.Listener = listener
	server.Logger.Log(lgr.Info, "Server is listening on %s", server.Address)

	for !server.ShuoldStop {
		server.Connection, err = server.Listener.Accept()
		if err != nil {
			server.Logger.Log(lgr.Error, "Error accepting connection: %s", err)
		}

		go server.ReceiveData()
	}
}

func (server *Server) Stop() {
	server.ShuoldStop = true
	server.Listener.Close()
	server.Logger.Log(lgr.Info, "Server stopped")
}

func (server *Server) SendData(event string, data string) {
	_, err := server.Connection.Write([]byte(Package{Event: event, Data: data}.String()))
	if err != nil {
		server.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	server.Logger.Log(lgr.Info, "Data sent: %s", data)
}

func (server *Server) ReceiveData() {
	data := make([]byte, 1024)
	n, err := server.Connection.Read(data)
	tmp := strings.Split(string(data[:n]), "|")
	if err != nil {
		server.Logger.Log(lgr.Error, "Error reading data: %s", err)
	}
	server.Logger.Log(lgr.Info, "Data received: %s", tmp[1])
	for _, event := range server.PossibleEvents {
		if event == tmp[0] {
			server.Events[tmp[0]](tmp[1])
			break
		}
	}
}

func (server *Server) On(event string, callback func(string)) {
	server.PossibleEvents = append(server.PossibleEvents, event)
	server.Events[event] = callback
}

func (client *Client) Connect() {
	conn, err := net.Dial("tcp", client.Address)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error connecting to server: %s", err)
	}
	client.Connection = conn
	client.Logger.Log(lgr.Info, "Connected to server")
}

func (client *Client) Disconnect() {
	client.ShouldStop = true
	client.Connection.Close()
	client.Logger.Log(lgr.Info, "Connection closed")
}

func (client *Client) SendData(event string, data string) {
	_, err := client.Connection.Write([]byte(Package{Event: event, Data: data}.String()))
	if err != nil {
		client.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data sent: %s", data)
}

func (client *Client) ReceiveData() {
	data := make([]byte, 1024)
	n, err := client.Connection.Read(data)
	tmp := strings.Split(string(data[:n]), "|")
	if err != nil {
		client.Logger.Log(lgr.Error, "Error receiving data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data received: %s", tmp[1])
	for _, event := range client.PossibleEvents {
		if event == tmp[0] {
			client.Events[tmp[0]](tmp[1])
			break
		}
	}
}

func (client *Client) On(event string, callback func(string)) {
	client.PossibleEvents = append(client.PossibleEvents, event)
	client.Events[event] = callback
}

func (client *Client) Listen() {
	for !client.ShouldStop {
		client.ReceiveData()
	}
}
