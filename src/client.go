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

	var message *WSMessage
	var err error
	for {
		message, err = c.NextMessageFrame()
		if err != nil {
			return nil, err
		}

		if message.Type() != Ping && message.Type() != Pong {
			break
		}

	}
	return message, err
}

func (c *WSClient) NextMessageFrame() (*WSMessage, error) {
	if c.connected == false {
		return nil, errors.New("connection is not open")
	}

	message := NewWSMessage(c)
	frame, err := message.NextWSFrame()
	if err != nil {
		return nil, err
	}
	message.frame = frame

	if message.frame.rsv1 || message.frame.rsv2 || message.frame.rsv3 {
		errCode := make([]byte, 2)
		binary.BigEndian.PutUint16(errCode, uint16(1002))
		SendMessage(c.conn, Close, errCode)
		return nil, errors.New("rsv fields must not be in use")
	}

	if message.Type() == Close {
		c.connected = false

		body, err := io.ReadAll(message)
		if err != nil {
			return nil, err
		}
		SendMessage(c.conn, Close, body)
		c.connected = false
		c.conn.Close()
		return message, nil
	}

	if message.Type() == Ping {
		body, err := io.ReadAll(message)
		if err != nil {
			return nil, err
		}

		if len(body) > 125 {
			errCode := make([]byte, 2)
			binary.BigEndian.PutUint16(errCode, uint16(1002))
			SendMessage(c.conn, Close, errCode)
			c.connected = false
			c.conn.Close()
		}
		SendMessage(c.conn, Pong, body)
	}

	if message.Type() == Pong {
		_, err := io.ReadAll(message)
		if err != nil {
			return nil, err
		}
	}

	return message, nil
}

func (c *WSClient) Close() error {

	SendMessage(c.conn, Close, []byte("OK"))
	//need to wait for a close frame response
	//and still process all remaining messages?

	c.connected = false
	c.conn.Close()
	return nil
}
