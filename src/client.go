package main

import (
	"bufio"
	"errors"
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
	Connect() error
	NextMessage() (*Message, error)
	Close()
}

func NewWSClient(addr string) *WSClient {
	return &WSClient{
		addr: addr,
	}
}

func (c *WSClient) Connect() error {
	c.connected = false
	conn, err := net.Dial("tcp", "127.0.0.1:9001")

	if err != nil {
		return err
	}
	c.conn = conn
	req, err := http.NewRequest("GET", "http://"+c.addr+"/getCaseCount", nil)
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

	message, err := NextWSMessage(c.reader)
	if err != nil {
		return nil, err
	}

	if message.Type() == Close {
		c.connected = false
		SendMessage(c.conn, Close, nil)
		c.conn.Close()
		return message, nil
	}

	return message, nil
}

func (c *WSClient) Close() error {
	SendMessage(c.conn, Close, nil)
	//need to wait for a close frame response
	c.connected = false
	defer c.conn.Close()
	return nil
}
