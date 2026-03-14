package main

import (
	"bufio"
	"errors"
	"io"
)

type Message interface {
	Type() Opcode
	ReadText() (string, error)
}

type WSMessage struct {
	reader *bufio.Reader
	frame  *WSFrame
}

func (m *WSMessage) Type() Opcode {
	return m.frame.opcode
}

func (m *WSMessage) ReadText() (string, error) {
	if m.frame.opcode != Text {
		return "", errors.New("invalid frame type")
	}
	buf := make([]byte, m.frame.length)
	_, err := io.ReadFull(m.frame.reader, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
