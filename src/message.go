package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"unicode/utf8"
)

type Message interface {
	Type() Opcode
	Read(p []byte) (n int, err error)
}

type WSMessage struct {
	client *WSClient
	frame  *WSFrame
	opcode Opcode
}

func NewWSMessage(client *WSClient) *WSMessage {
	return &WSMessage{
		client: client,
	}
}

func (m *WSMessage) Type() Opcode {
	return m.opcode
}

func (m *WSMessage) Read(p []byte) (n int, err error) {
	for len(p) > 0 {
		frameN, err := ReadFrame(m.frame, m.client.connReader, p, len(p))
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
				err = NextMessageFrame(m)
				if err != nil {
					return n, err
				}
				// fmt.Println("Read - opcode", m.frame.opcode)
				// fmt.Println("Read - body length", m.frame.length)
				if m.frame.opcode != Ping && m.frame.opcode != Pong {
					break
				}
			}

			if m.frame.opcode != Continuation {
				m.client.CloseWithError()
				return n, errors.New("expected continuation frame")
			}
			continue
		}

		return n, nil
	}

	return n, nil
}

func NextWSMessage(client *WSClient) (*WSMessage, error) {
	message := NewWSMessage(client)
	var err error
	for {
		err = NextMessageFrame(message)
		if err != nil {
			return nil, err
		}

		if message.frame.opcode != Ping && message.frame.opcode != Pong {
			break
		}
	}
	message.opcode = message.frame.opcode
	return message, nil
}

func NextMessageFrame(message *WSMessage) error {
	if message.client.connected == false {
		return io.EOF
	}

	frame, err := ReadWSFrame(message.client.connReader)
	if err != nil {
		return err
	}

	// fmt.Println("NextMessageFrame - opcode", frame.opcode)
	// fmt.Println("NextMessageFrame - body length", frame.length)
	message.frame = frame

	if message.frame.rsv1 || message.frame.rsv2 || message.frame.rsv3 {
		message.client.CloseWithError()
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
		message.client.CloseWithError()
	}

	if message.frame.opcode == Close {
		if !message.frame.final {
			message.client.CloseWithError()
		}

		body, err := io.ReadAll(message)
		if err != nil {
			return err
		}

		if len(body) == 1 {
			message.client.CloseWithError()
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
				message.client.CloseWithError()
				return nil
			}
		}

		WriteMessage(message.client.conn, Close, body)
		message.client.connected = false
		message.client.conn.Close()
		return nil
	}

	if message.frame.opcode == Ping {
		if !message.frame.final {
			message.client.CloseWithError()
		}

		body, err := io.ReadAll(message)
		if err != nil {
			return err
		}

		if len(body) > 125 {
			message.client.CloseWithError()
		}
		WriteMessage(message.client.conn, Pong, body)
	}

	if message.frame.opcode == Pong {
		if !message.frame.final {
			message.client.CloseWithError()
		}

		_, err := io.ReadAll(message)
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteMessage(conn net.Conn, opcode Opcode, body []byte) error {
	frame := NewWSFrame(true)
	frame.final = true
	frame.opcode = opcode
	frame.length = uint64(len(body))

	err := frame.Write(conn, body)
	return err
}
