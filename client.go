package tcp

import (
	"net"
	"strings"
	"time"

	lgr "github.com/antosmichael07/Go-Logger"
)

type Client struct {
	// The protocol to use
	Protocol string
	// Connection to the server
	Connection net.Conn
	// Address of the server
	Address string
	// Logger is the logger of the client, can be customized (https://github.com/antosmichael07/Go-Logger)
	Logger lgr.Logger
	// Here are the events that the client can handle
	Events         [65536]func(*[]byte)
	PossibleEvents []uint16
	// ShouldStop is a boolean that is used to stop the client
	ShouldStop bool
	// Token is the token of the client
	Token [64]byte
	// OnConnectFunc is the function that is called when the client connects if IsOnConnect is true
	OnConnectFunc func()
	IsOnConnect   bool
}

// toByte is a function that converts the package to a byte array to be sent
func (pkg Package) toByteClient(token *[64]byte) (data []byte) {
	data = make([]byte, 74+len(pkg.Data))
	size := uint64(74 + len(pkg.Data))

	for i := 0; i < 8; i++ {
		data[i] = byte(size >> (8 * i))
	}
	for i := 0; i < 64; i++ {
		data[i+8] = token[i]
	}
	for i := 0; i < 2; i++ {
		data[i+72] = byte(pkg.Event >> (8 * i))
	}
	for i := 0; i < len(pkg.Data); i++ {
		data[i+74] = pkg.Data[i]
	}

	return data
}

// NewClient is a function that creates a new client with the given address
func NewClient(address, protocol string) Client {
	logger, _ := lgr.NewLogger(strings.ToUpper(protocol), "", false)

	return Client{
		Protocol:       protocol,
		Connection:     nil,
		Address:        address,
		Logger:         logger,
		Events:         [65536]func(*[]byte){},
		PossibleEvents: []uint16{},
		ShouldStop:     false,
		Token:          [64]byte{},
		OnConnectFunc:  func() {},
		IsOnConnect:    false,
	}
}

// Connect is a function that connects the client to the server
func (client *Client) Connect() error {
	// Connect to the server
	net_dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := net_dialer.Dial(client.Protocol, client.Address)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error connecting to server: %s", err)
		return err
	}
	// Save the connection
	client.Connection = conn
	client.Logger.Log(lgr.Info, "Connected to server")

	// Receive the token
	client.On(event_token, func(data *[]byte) {
		client.Logger.Log(lgr.Info, "Token received: %v", data)
		// Save the token
		client.Token = [64]byte(*data)
		// Tell the server that the data was received
		client.SendData(event_last_data_received, &[]byte{})
		// Call the OnConnect function
		if client.IsOnConnect {
			go client.OnConnectFunc()
		}
	})

	client.On(event_are_you_ok, func(data *[]byte) {
		client.SendData(event_are_you_ok, &[]byte{})
		time.Sleep(1 * time.Second)
		client.SendData(event_are_you_ok, &[]byte{})
		time.Sleep(1 * time.Second)
		client.SendData(event_are_you_ok, &[]byte{})
		time.Sleep(1 * time.Second)
		client.SendData(event_are_you_ok, &[]byte{})
	})

	return nil
}

// Listen is a function that listens for data from the server
// Should be called after you have set the events
func (client *Client) Listen() {
	client.Logger.Log(lgr.Info, "Started listening")

	// Send the connect event
	client.SendData(event_connect, &[]byte{})

	for !client.ShouldStop {
		client.receiveData()
	}
}

// Disconnect is a function that disconnects the client from the server
func (client *Client) Disconnect() {
	client.ShouldStop = true
	client.Connection.Close()
	client.Logger.Log(lgr.Info, "Connection closed")
}

// SendData is a function that sends data to the server with the given event name, and data
func (client *Client) SendData(event uint16, data *[]byte) {
	// Convert the package to a byte array
	to_send := Package{Event: event, Data: *data}.toByteClient(&client.Token)
	// Send the data
	_, err := client.Connection.Write(to_send)
	if err != nil {
		client.Logger.Log(lgr.Error, "Error sending data: %s", err)
	}
	client.Logger.Log(lgr.Info, "Data sent with the event name: %v", event)
}

// receiveData is a function that receives data from the server
func (client *Client) receiveData() {
	// Read the data
	data := make([]byte, 16384)
	n, err := client.Connection.Read(data)
	if client.Token != [64]byte{} {
		if !client.ShouldStop {
			// Tell the server that the data was received
			client.SendData(event_last_data_received, &[]byte{})
		}
	}
	data = data[:n]
	if err != nil {
		client.Logger.Log(lgr.Error, "Error reading received data: %s", err)
		if !client.ShouldStop {
			client.Disconnect()
		}
		return
	}

	// Decode the data
	pkg := Package{}
	pkg.fromByte(&data)
	if pkg.Size > uint64(len(data)) {
		client.Logger.Log(lgr.Error, "Invalid data received: %v", data)
		return
	}
	pkg.Data = pkg.Data[:pkg.Size-74]

	// If the event is valid, call the function that is associated with the event
	client.Logger.Log(lgr.Info, "Data received with an event name: %v, data: %v", pkg.Event, pkg.Data)
	for _, event := range client.PossibleEvents {
		if event == pkg.Event {
			go client.Events[pkg.Event](&pkg.Data)
			break
		}
	}
}

// On is a function that adds an event to the client
func (client *Client) On(event uint16, callback func(*[]byte)) {
	client.PossibleEvents = append(client.PossibleEvents, event)
	client.Events[event] = callback
}

// OnConnect is a function that sets the OnConnectFunc and IsOnConnect to true
func (client *Client) OnConnect(callback func()) {
	client.OnConnectFunc = callback
	client.IsOnConnect = true
}
