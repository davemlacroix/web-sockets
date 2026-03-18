package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
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

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "upgrade")
	req.Header.Add("Sec-WebSocket-Key", key)
	req.Header.Add("Origin", "127.0.0.1")
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

	// Validate Sec-WebSocket-Accept per RFC 6455 4.1
	const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + guid))
	expectedAccept := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if resp.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		return errors.New("invalid Sec-WebSocket-Accept header")
	}

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

	if message.Type() == Continuation {
		c.CloseWithError(1002)
		return 0, io.EOF
	}

	if message.Type() == Text || message.Type() == Binary {
		body, err := io.ReadAll(message)

		if message.Type() == Text {
			if !utf8.Valid(body) {
				c.CloseWithError(1007)
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

func (c *WSClient) CloseWithError(code uint16) {
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, code)
	WriteMessage(c.conn, Close, errCode)
	c.connected = false
	c.conn.Close()
}
