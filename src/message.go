package main

import (
	"errors"
	"io"
	"net"
	"unicode/utf8"
)

type Message interface {
	Type() Opcode
	Read(p []byte) (n int, err error)
	ReadText() (string, error)
}

type WSMessage struct {
	client *WSClient
	frame  *WSFrame
	opcode Opcode
}

func (m *WSMessage) Type() Opcode {
	return m.opcode
}

func (m *WSMessage) Read(p []byte) (n int, err error) {
	for len(p) > 0 {
		frameN, err := ReadFrame(m, p, len(p))
		n += frameN

		if err != nil && err != io.EOF {
			return n, err
		}

		if m.frame.opcode == Close {
			if frameN >= 2 && !utf8.Valid(p[2:frameN]) {
				m.client.CloseWithError()
				return n, errors.New("invalid utf8")
			}
		}

		p = p[frameN:]

		if err == io.EOF {
			if m.frame.final {
				return n, io.EOF
			}
			for {
				if err = m.client.NextMessageFrame(m); err != nil {
					return n, err
				}
				// fmt.Println("Read - opcode", m.frame.opcode)
				// fmt.Println("Read - body length", m.frame.length)
				if m.frame.opcode != Ping && m.frame.opcode != Pong {
					break
				}
			}

			if m.frame.opcode != Continuation {
				return n, errors.New("expected continuation frame")
			}
			continue
		}

		return n, nil
	}

	return n, nil
}

func ReadFrame(m *WSMessage, p []byte, readLen int) (n int, err error) {
	if m.frame == nil {
		return 0, errors.New("no frame available")
	}

	if m.frame.payloadRemaining == 0 {
		return 0, io.EOF
	}

	if uint64(readLen) > m.frame.payloadRemaining {
		readLen = int(m.frame.payloadRemaining)
	}

	n, err = io.ReadFull(m.client.connReader, p[:readLen])
	if err != nil {
		return n, err
	}

	m.frame.payloadRemaining -= uint64(n)
	if m.frame.payloadRemaining == 0 {
		return n, io.EOF
	}

	return n, nil
}

func NewWSMessage(client *WSClient) *WSMessage {
	return &WSMessage{
		client: client,
	}
}

func (m *WSMessage) NextWSFrame() (*WSFrame, error) {
	frame, err := ReadWSFrame(m.client.connReader)
	if err != nil {
		return nil, err
	}
	return frame, err
}

func WriteMessage(conn net.Conn, opcode Opcode, body []byte) error {
	frame := NewWSFrame(true)
	frame.final = true
	frame.opcode = opcode
	frame.length = uint64(len(body))

	err := frame.Write(conn, body)
	return err
}
