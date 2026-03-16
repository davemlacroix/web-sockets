package main

import (
	"bufio"
	"errors"
	"io"
	"net"
)

type Message interface {
	Type() Opcode
	Read(p []byte) (n int, err error)
	ReadText() (string, error)
}

type WSMessage struct {
	reader           *bufio.Reader
	frame            *WSFrame
	payloadRemaining uint64
}

func (m *WSMessage) Type() Opcode {
	return m.frame.opcode
}

func (m *WSMessage) Read(p []byte) (n int, err error) {
	// this will need to be changed to work for multiple frames
	// in a single message
	if m.frame == nil {
		return 0, errors.New("no frame available")
	}

	if m.payloadRemaining == 0 {
		return 0, io.EOF
	}

	readLen := len(p)
	if uint64(readLen) > m.payloadRemaining {
		readLen = int(m.payloadRemaining)
	}

	n, err = io.ReadFull(m.reader, p[:readLen])
	if err != nil {
		return n, err
	}

	m.payloadRemaining -= uint64(n)
	if m.payloadRemaining == 0 {
		return n, io.EOF
	}

	return n, nil
}

func (m *WSMessage) ReadText() (string, error) {
	if m.frame.opcode != Text {
		return "", errors.New("invalid frame type")
	}

	buf := make([]byte, 4096)
	text := ""
	for {
		n, err := m.Read(buf)

		if n > 0 {
			text += string(buf[:n])
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return text, nil
}

func NextWSMessage(reader *bufio.Reader) (*WSMessage, error) {
	frame, err := ReadWSFrame(reader)
	if err != nil {
		return nil, err
	}
	// fmt.Println("Frame Header ---------------------")
	// fmt.Println("final: ", frame.final)
	// fmt.Println("opcode: ", frame.opcode)
	// fmt.Println("masked: ", frame.masked)
	// fmt.Println("length: ", frame.length)
	// fmt.Println("End Frame Header -----------------")

	m := &WSMessage{
		reader:           reader,
		frame:            frame,
		payloadRemaining: frame.length,
	}
	return m, nil
}

func SendMessage(conn net.Conn, opcode Opcode, body []byte) {
	frame := NewWSFrame(true)
	frame.final = true
	frame.opcode = opcode
	frame.length = 0

	frame.Write(conn, body)
}
