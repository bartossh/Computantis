package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

const (
	hubInnerChannelsBufferSize      = 100
	socketContextExp                = 5 * time.Second
	socketWriteWait                 = 10 * time.Second
	socketPongWait                  = 20 * time.Second
	socketPingPeriod                = (socketPongWait * 4) / 5
	socketReadBufferSize            = 5012
	socketWriteBufferSize           = socketReadBufferSize * 256
	socketMaxMessageSize            = socketWriteBufferSize * 4
	clientMessageChannelsBufferSize = 512
)

const (
	echo = "echo"
)

// UpgradeConnectionRequest is a request to upgrade to websocket.
// Request contains signed Data previously sent to client.
// Signature verifies if given Address is paired with private key that
// was used to sign the data.
type UpgradeConnectionRequest struct {
	Address   string   `json:"address"`
	Token     string   `json:"token"`
	Data      []byte   `json:"data"`
	Hash      [32]byte `json:"hash"`
	Signature []byte   `json:"signature"`
}

// Message is the message that is used to exchange information between
// the server and the client.
type Message struct {
	receivers []string
	Command   string `json:"command"` // Command is the command that refers to the action handler in websocket protocol.
	Error     string `json:"error"`   // Error is the error message that is sent to the client.
	Data      []byte `json:"data"`    // Data is compressed data that is sent to the client. Based on the command, the client will know how to decompress the data.
}

type socket struct {
	address string
	hub     *hub
	conn    *websocket.Conn
	send    chan []byte
	repo    Repository
	close   chan struct{}
	log     logger.Logger
}

func (s *server) wsWrapper(c *fiber.Ctx) error {
	var req UpgradeConnectionRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok, err := s.repo.CheckToken(c.Context(), req.Token); !ok || err != nil {
		s.log.Error(fmt.Sprintf("failed to check token: %s", err.Error()))
		return fiber.ErrForbidden
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("failed to validate data for address %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("failed to verify signature: %s", err.Error()))
		return fiber.ErrForbidden
	}

	client := &socket{
		address: req.Address,
		hub:     s.hub,
		conn:    nil,
		send:    make(chan []byte, clientMessageChannelsBufferSize),
		repo:    s.repo,
		close:   make(chan struct{}, 1),
		log:     s.log,
	}

	serveWs := func(conn *websocket.Conn) {
		client.conn = conn
		client.hub.register <- client
		go client.writePump()
		client.readPump()
	}

	return websocket.New(serveWs)(c)
}

func (c *socket) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(socketMaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(socketPongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(socketPongWait)); return nil })
	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			switch {
			case websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure):
				c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to unexpected error %s\n", c.address, err))
			default:
				c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to error %s\n", c.address, err))
			}

			close(c.send)
			break
		}
		c.process(&msg)
	}
}

func (c *socket) writePump() {
	ticker := time.NewTicker(socketPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case raw, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(socketWriteWait))
			if !ok {
				c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to channel close", c.address))
				c.hub.unregister <- c
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				c.log.Error(fmt.Sprintf("socket closing connection to the client %s due to %s", c.address, err))
				c.hub.unregister <- c
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(socketWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(c.address)); err != nil {
				c.log.Error(fmt.Sprintf("socket closing connection to the client %s due to %s", c.address, err))
				c.hub.unregister <- c
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
		}
	}
}

func ctxClose(close <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), socketContextExp)
	go func() {
	outer:
		for {
			select {
			case <-close:
				cancel()
				break outer
			case <-ctx.Done():
				break outer
			}
		}
	}()
	return ctx, cancel
}

type hub struct {
	clients    map[string]*socket
	broadcast  chan *Message
	register   chan *socket
	unregister chan *socket
	log        logger.Logger
}

func newHub(log logger.Logger) *hub {
	return &hub{
		broadcast:  make(chan *Message, hubInnerChannelsBufferSize),
		register:   make(chan *socket, hubInnerChannelsBufferSize),
		unregister: make(chan *socket, hubInnerChannelsBufferSize),
		clients:    make(map[string]*socket, hubInnerChannelsBufferSize),
		log:        log,
	}
}

func (h *hub) run(ctx context.Context) {
outer:
	for {
		select {
		case client := <-h.register:
			h.clients[client.address] = client
		case client := <-h.unregister:
			delete(h.clients, client.address)
		case message := <-h.broadcast:
			raw, err := json.Marshal(&message)
			if err != nil {
				h.log.Error(fmt.Sprintf("hub failed to marshal message: %s", err.Error()))
				continue outer
			}
			for _, addr := range message.receivers {
				client, ok := h.clients[addr]
				if !ok {
					continue outer
				}
				select {
				case h.clients[addr].send <- raw:
				default:
					delete(h.clients, client.address)
				}
			}
		case <-ctx.Done():
			for _, client := range h.clients {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				delete(h.clients, client.address)
				close(client.close)
			}
			break outer
		}
	}
}

func (c *socket) process(msg *Message) {
	ctx, cancel := ctxClose(c.close)
	defer cancel()
	switch msg.Command {
	case echo:
		if err := c.echo(ctx, msg); err != nil {
			c.sendCommand(setCommandError(msg, err))
		}
		c.sendCommand(msg)

	default:
		c.log.Info(fmt.Sprintf("socket received unknown command %s", msg.Command))
		c.sendCommand(setCommandError(msg, fmt.Errorf("unknown command %s", msg.Command)))
	}
}

func setCommandError(msg *Message, err error) *Message {
	msg.Error = err.Error()
	return msg
}

func (c socket) sendCommand(msg *Message) {
	raw, err := json.Marshal(&msg)
	if err != nil {
		c.log.Error(fmt.Sprintf("socket failed to marshal message: %s", err.Error()))
		return
	}
	c.send <- raw
}

func (c socket) broadcastCommend(msg *Message) {
	c.hub.broadcast <- msg
}

func (c *socket) echo(_ context.Context, msg *Message) error {
	return nil
}
