package tcp

import (
	"encoding/json"
	"net"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	Connection     net.Conn
	Listener       net.Listener
	Address        string
	Logger         lgr.Logger
	Events         map[string]func([]byte)
	PossibleEvents []string
	ShuoldStop     bool
}

type Client struct {
	Connection     net.Conn
	Address        string
	Logger         lgr.Logger
	Events         map[string]func([]byte)
	PossibleEvents []string
	ShouldStop     bool
}

type Package struct {
	Event string
	Data  []byte
}

func (pkg Package) ToByte() []byte {
	logger := lgr.NewLogger("TCP")

	data, err := json.Marshal(pkg)
	if err != nil {
		logger.Log(lgr.Error, "Error marshaling package")
		panic(err)
	}

	return data
}

func NewServer(address string) Server {
	return Server{
		Connection:     nil,
		Listener:       nil,
		Address:        address,
		Logger:         lgr.NewLogger("TCP"),
		Events:         make(map[string]func([]byte)),
		PossibleEvents: []string{},
		ShuoldStop:     false,
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

func (server *Server) SendData(event string, data []byte) {
	_, err := server.Connection.Write(Package{Event: event, Data: data}.ToByte())
	if err != nil {
		server.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	server.Logger.Log(lgr.Info, "Data sent: %s", data)
}

func (server *Server) ReceiveData() {
	data := make([]byte, 1024)
	_, err := server.Connection.Read(data)
	if err != nil {
		server.Logger.Log(lgr.Error, "Error reading data: %s", err)
		return
	}

	pkg := Package{}
	err = json.Unmarshal(data, &pkg)
	if err != nil {
		server.Logger.Log(lgr.Error, "Error unmarshaling package: %s", err)
		return
	}

	server.Logger.Log(lgr.Info, "Data received with an event name: %s", pkg.Event)
	for _, event := range server.PossibleEvents {
		if event == pkg.Event {
			server.Events[pkg.Event](pkg.Data)
			break
		}
	}
}

func (server *Server) On(event string, callback func([]byte)) {
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

func (client *Client) SendData(event string, data []byte) {
	_, err := client.Connection.Write(Package{Event: event, Data: data}.ToByte())
	if err != nil {
		client.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data sent: %s", data)
}

func (client *Client) ReceiveData() {
	data := make([]byte, 1024)
	_, err := client.Connection.Read(data)
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
