package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
)

type WSClient struct {
	connected bool
	addr      string
	conn      net.Conn
	reader    *bufio.Reader
}

type Client interface {
	Connect(urlPath string) error
	NextMessage() (*Message, error)
	Close()
}

func NewWSClient(addr string) *WSClient {
	return &WSClient{
		addr: addr,
	}
}

func (c *WSClient) Connect(urlPath string) error {
	c.connected = false
	conn, err := net.Dial("tcp", "127.0.0.1:9001")

	if err != nil {
		return err
	}
	c.conn = conn
	req, err := http.NewRequest("GET", "http://"+c.addr+urlPath, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "upgrade")
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Add("Origin", "127.0.0.1")
	req.Header.Add("Sec-WebSocket-Protocol", "chat, superchat")
	req.Header.Add("Sec-WebSocket-Version", "13")

	err = req.Write(c.conn)
	if err != nil {
		return err
	}

	c.reader = bufio.NewReader(c.conn)
	resp, err := http.ReadResponse(c.reader, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.connected = true
	return nil
}

func (c *WSClient) NextMessage() (*WSMessage, error) {
	if c.connected == false {
		return nil, errors.New("connection is not open")
	}

	message := NewWSMessage(c)
	var err error
	for {
		err = c.NextMessageFrame(message)
		if err != nil {
			return nil, err
		}

		if message.frame.opcode != Ping && message.frame.opcode != Pong {
			break
		}
	}
	message.opcode = message.frame.opcode
	return message, err
}

func (c *WSClient) NextMessageFrame(message *WSMessage) error {
	if c.connected == false {
		return errors.New("connection is not open")
	}

	frame, err := message.NextWSFrame()
	if err != nil {
		return err
	}
	// fmt.Println("NextMessageFrame - opcode", frame.opcode)
	// fmt.Println("NextMessageFrame - body length", frame.length)
	message.frame = frame

	if message.frame.rsv1 || message.frame.rsv2 || message.frame.rsv3 {
		c.CloseWithError()
		return errors.New("rsv fields must not be in use")
	}

	switch message.frame.opcode {
	case Continuation:
	case Text:
	case Binary:
	case Close:
	case Ping:
	case Pong:
	default:
		c.CloseWithError()
	}

	if message.frame.opcode == Close {
		c.connected = false

		body, err := io.ReadAll(message)
		if err != nil {
			return err
		}
		SendMessage(c.conn, Close, body)
		c.connected = false
		c.conn.Close()
		return nil
	}

	if message.frame.opcode == Ping {
		if !message.frame.final {
			c.CloseWithError()
		}

		body, err := io.ReadAll(message)
		if err != nil {
			return err
		}

		if len(body) > 125 {
			c.CloseWithError()
		}
		SendMessage(c.conn, Pong, body)
	}

	if message.frame.opcode == Pong {
		_, err := io.ReadAll(message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *WSClient) Close() error {

	SendMessage(c.conn, Close, []byte("OK"))
	//need to wait for a close frame response
	//and still process all remaining messages?

	c.connected = false
	c.conn.Close()
	return nil
}

func (c *WSClient) CloseWithError() {
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(1002))
	SendMessage(c.conn, Close, errCode)
	c.connected = false
	c.conn.Close()
}
