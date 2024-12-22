package tcp

// Package is a struct that contains the token, the event and the data that is sent
type Package struct {
	Size  uint64
	Token [64]byte
	Event uint16
	Data  []byte
}

const (
	event_last_data_received uint16 = iota
	event_connect
	event_token
	event_are_you_ok
)
