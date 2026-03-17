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
	reader        *bufio.Reader
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

	c.reader = bufio.NewReader(c.conn)
	resp, err := http.ReadResponse(c.reader, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.connected = true
	return nil
}

func (c *WSClient) Close() error {

	WriteMessage(c.conn, Close, []byte("OK"))

	c.connected = false
	c.conn.Close()
	return nil
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

	message := NewWSMessage(c)
	var err error
	for {
		err = c.NextMessageFrame(message)
		if err != nil {
			return 0, err
		}

		if message.frame.opcode != Ping && message.frame.opcode != Pong {
			break
		}
	}
	message.opcode = message.frame.opcode

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

func (c *WSClient) NextMessageFrame(message *WSMessage) error {
	if c.connected == false {
		return io.EOF
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

		if len(body) == 1 {
			c.CloseWithError()
			return nil
		}

		if len(body) >= 2 {
			code := binary.BigEndian.Uint16(body[:2])
			invalid := false
			switch {
			case code == 1000, code == 1001, code == 1002, code == 1003,
				code == 1007, code == 1008, code == 1009, code == 1010, code == 1011:
			case code >= 3000 && code <= 4999:
			default:
				invalid = true
			}
			if invalid {
				c.CloseWithError()
				return nil
			}
		}

		WriteMessage(c.conn, Close, body)
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
		WriteMessage(c.conn, Pong, body)
	}

	if message.frame.opcode == Pong {
		_, err := io.ReadAll(message)
		if err != nil {
			return err
		}
	}

	return nil
}
