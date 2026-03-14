package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

type WSClient struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader
}

type Client interface {
	Connect() error
	ReadFrame() (*Frame, error)
	Close()
}

func NewWSClient(addr string) *WSClient {
	return &WSClient{
		addr: addr,
	}
}

func (c *WSClient) Connect() error {
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

	return nil
}

func (c *WSClient) ReadFrame() (*WSFrame, error) {
	frame, err := NextWSFrame(c.reader)
	if err != nil {
		return nil, err
	}

	fmt.Println("final: ", frame.final)
	fmt.Println("opcode: ", frame.opcode)
	fmt.Println("masked: ", frame.masked)
	fmt.Println("length: ", frame.length)

	buf := make([]byte, 1024)
	io.ReadFull(c.reader, buf)
	fmt.Println(buf)
	return frame, nil
}

func (c *WSClient) Close() error {
	defer c.conn.Close()
	return nil
}
