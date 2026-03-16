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
	reader *bufio.Reader
	frame  *WSFrame
}

func (m *WSMessage) Type() Opcode {
	return m.frame.opcode
}

// func (m *WSMessage) Read(p []byte) (n int, err error) {
// 	// this will need to be changed to work for multiple frames
// 	// in a single message
// 	n = 0
// 	for {
// 		fmt.Println("frame read ------")
// 		frameN, err := m.ReadFrame(p)
// 		n += frameN
// 		fmt.Println(n)
// 		if err != nil {
// 			fmt.Println("frame read error ------")
// 			fmt.Println(err)
// 			return n, err
// 		}

// 		if m.frame.final {
// 			break
// 		}

// 		m.frame, err = ReadWSFrame(m.reader)
// 	}

// 	return n, err
// }

func (m *WSMessage) Read(p []byte) (n int, err error) {
	if m.frame == nil {
		return 0, errors.New("no frame available")
	}

	if m.frame.payloadRemaining == 0 {
		return 0, io.EOF
	}

	readLen := len(p)
	if uint64(readLen) > m.frame.payloadRemaining {
		readLen = int(m.frame.payloadRemaining)
	}

	n, err = io.ReadFull(m.reader, p[:readLen])
	if err != nil {
		return n, err
	}

	m.frame.payloadRemaining -= uint64(n)
	if m.frame.payloadRemaining == 0 {
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
		reader: reader,
		frame:  frame,
	}
	return m, nil
}

func SendMessage(conn net.Conn, opcode Opcode, body []byte) {
	frame := NewWSFrame(true)
	frame.final = true
	frame.opcode = opcode
	frame.length = uint64(len(body))

	frame.Write(conn, body)
}
