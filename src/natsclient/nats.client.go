package natsclient

import (
	"errors"
	"net/url"

	"github.com/nats-io/nats.go"
)

const (
	PubSubAwaitingTrxs string = "awaiting_trxs_for_addresses"
)

var ErrEmptyAddressProvided = errors.New("nats server address is empty")

// Config contains all arguments required to connect to the nats service
type Config struct {
	Address string `yaml:"server_address"`
	Name    string `yaml:"client_name"`
	Token   string `yaml:"token"`
}

type socket struct {
	conn *nats.Conn
}

func connect(cfg Config) (*socket, error) {
	if cfg.Address == "" {
		return nil, ErrEmptyAddressProvided
	}
	_, err := url.Parse(cfg.Address)
	if err != nil {
		return nil, err
	}
	var s socket
	s.conn, err = nats.Connect(cfg.Address, nats.Name(cfg.Name), nats.Token(cfg.Token))
	return &s, err
}

// Disconnect drains the message queue and disconnects from the pub/sub.
// Nats Drain will put a connection into a drain state.
// All subscriptions will immediately be put into a drain state.
// Upon completion, the publishers will be drained and can not publish any additional messages.
func (s *socket) Disconnect() error {
	return s.conn.Drain()
}
