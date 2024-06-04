package tcp

import (
	"net"
	"strings"

	custom_logger "github.com/antosmichael07/Go-Logger"
)

type Server struct {
	Connection net.Conn
	Listener   net.Listener
	Address    string
	Logger     custom_logger.Logger
	Events     map[string]func(string)
}

type Client struct {
	Connection net.Conn
	Address    string
	Logger     custom_logger.Logger
	Events     map[string]func(string)
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
		Connection: nil,
		Listener:   nil,
		Address:    address,
		Logger:     custom_logger.NewLogger(),
		Events:     make(map[string]func(string)),
	}
}

func NewClient(address string) Client {
	return Client{
		Connection: nil,
		Address:    address,
		Logger:     custom_logger.NewLogger(),
		Events:     make(map[string]func(string)),
	}
}

func (server *Server) Start() {
	listener, err := net.Listen("tcp", server.Address)
	if err != nil {
		panic(err)
	}
	server.Listener = listener
	server.Logger.Log("Server is listening on %s", server.Address)

	for {
		server.Connection, err = server.Listener.Accept()
		if err != nil {
			server.Logger.Log("Error accepting connection: %s", err)
		}

		go server.ReceiveData()
	}
}

func (server *Server) Stop() {
	server.Listener.Close()
	server.Logger.Log("Server stopped")
}

func (server *Server) SendData(event string, data string) {
	_, err := server.Connection.Write([]byte(Package{Event: event, Data: data}.String()))
	if err != nil {
		server.Logger.Log("Error sending data: %s", err)
	}
	server.Logger.Log("Data sent: %s", data)
}

func (server *Server) ReceiveData() {
	data := make([]byte, 1024)
	n, err := server.Connection.Read(data)
	tmp := strings.Split(string(data[:n]), "|")
	if err != nil {
		server.Logger.Log("Error reading data: %s", err)
	}
	server.Logger.Log("Data received: %s", tmp[1])
	server.Events[tmp[0]](tmp[1])
}

func (server *Server) On(event string, callback func(string)) {
	server.Events[event] = callback
}

func (client *Client) Connect() {
	conn, err := net.Dial("tcp", client.Address)
	if err != nil {
		client.Logger.Log("Error connecting to server: %s", err)
	}
	client.Connection = conn
	client.Logger.Log("Connected to server")

	go client.ReceiveData()
}

func (client *Client) Disconnect() {
	client.Connection.Close()
	client.Logger.Log("Connection closed")
}

func (client *Client) SendData(event string, data string) {
	_, err := client.Connection.Write([]byte(Package{Event: event, Data: data}.String()))
	if err != nil {
		client.Logger.Log("Error sending data: %s", err)
	}
	client.Logger.Log("Data sent: %s", data)
}

func (client *Client) ReceiveData() {
	data := make([]byte, 1024)
	n, err := client.Connection.Read(data)
	tmp := strings.Split(string(data[:n]), "|")
	if err != nil {
		client.Logger.Log("Error receiving data: %s", err)
	}
	client.Logger.Log("Data received: %s", tmp[1])
	client.Events[tmp[0]](tmp[1])
}

func (client *Client) On(event string, callback func(string)) {
	client.Events[event] = callback
}
