package main

import (
	"bufio"
	"errors"
	"fmt"
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

func NextWSMessage(reader *bufio.Reader) (*WSMessage, error) {
	frame, err := NextWSFrame(reader)
	if err != nil {
		return nil, err
	}
	fmt.Println("Frame Header ---------------------")
	fmt.Println("final: ", frame.final)
	fmt.Println("opcode: ", frame.opcode)
	fmt.Println("masked: ", frame.masked)
	fmt.Println("length: ", frame.length)
	fmt.Println("End Frame Header -----------------")

	m := &WSMessage{
		reader: reader,
		frame:  frame,
	}
	return m, nil
}
