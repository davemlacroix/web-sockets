package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"unicode/utf8"
)

type WSClient struct {
	connected     bool
	addr          string
	conn          net.Conn
	connReader    *bufio.Reader
	messageReady  bool
	message       []byte
	messageReader *bufio.Reader
}

type Client interface {
	Connect(urlPath string) error
	Close()
	NextMessage() (Opcode, error)
	Read(p []byte) (n int, err error)
	Write(opcode Opcode, body []byte) error
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

	c.connReader = bufio.NewReader(c.conn)
	resp, err := http.ReadResponse(c.connReader, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.connected = true
	return nil
}

func (c *WSClient) Close() {
	WriteMessage(c.conn, Close, []byte("OK"))

	c.connected = false
	c.conn.Close()
}

func (c *WSClient) Read(p []byte) (n int, err error) {
	if !c.messageReady {
		return 0, errors.New("no message ready")
	}

	n, err = c.messageReader.Read(p)
	if err == io.EOF {
		c.messageReady = false
	}
	return n, err
}

func (c *WSClient) Write(opcode Opcode, body []byte) error {
	err := WriteMessage(c.conn, opcode, body)
	return err
}

func (c *WSClient) NextMessage() (Opcode, error) {
	if c.connected == false {
		return 0, io.EOF
	}

	message, err := NextWSMessage(c)
	if err != nil {
		return 0, err
	}

	if message.Type() == Text || message.Type() == Binary {
		body, err := io.ReadAll(message)

		if message.Type() == Text {
			if !utf8.Valid(body) {
				c.CloseWithError()
				return 0, err
			}
		}
		c.message = body
		c.messageReader = bufio.NewReader(bytes.NewReader(body))

		if err != nil && err != io.EOF {
			return 0, err
		}
		c.messageReady = true
	} else {
		return 0, io.EOF
	}

	return message.Type(), nil
}

func (c *WSClient) CloseWithError() {
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(1002))
	WriteMessage(c.conn, Close, errCode)
	c.connected = false
	c.conn.Close()
}
