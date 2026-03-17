package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
)

type Opcode int

const (
	Continuation Opcode = 0
	Text         Opcode = 1
	Binary       Opcode = 2
	Close        Opcode = 8
	Ping         Opcode = 9
	Pong         Opcode = 10
)

type Frame interface {
	Write(conn net.Conn, content []byte) error
}

type WSFrame struct {
	final            bool
	rsv1             bool
	rsv2             bool
	rsv3             bool
	opcode           Opcode
	masked           bool
	length           uint64
	mask             [4]byte
	payloadRemaining uint64
}

func NewWSFrame(masked bool) *WSFrame {
	var mask [4]byte
	if masked {
		rand.Read(mask[:])
	}

	return &WSFrame{
		masked: masked,
		mask:   mask,
	}
}

func (f *WSFrame) Write(conn net.Conn, body []byte) error {
	var frame []byte
	b1 := byte(0)
	if f.final {
		b1 |= 0x80
	}
	b1 |= byte(f.opcode) & 0x0F
	frame = append(frame, b1)

	b2 := byte(0)
	if f.masked {
		b2 |= 0x80
	}

	if f.length <= 125 {
		b2 |= byte(f.length)
		frame = append(frame, b2)
	} else if f.length <= 0xFFFF {
		b2 |= 126
		frame = append(frame, b2)
		extLen := make([]byte, 2)
		binary.BigEndian.PutUint16(extLen, uint16(f.length))
		frame = append(frame, extLen...)
	} else {
		b2 |= 127
		frame = append(frame, b2)
		extLen := make([]byte, 8)
		binary.BigEndian.PutUint64(extLen, uint64(f.length))
		frame = append(frame, extLen...)
	}

	if f.masked {
		frame = append(frame, f.mask[:]...)
		masked := make([]byte, f.length)
		copy(masked, body)

		for i := uint64(0); i < f.length; i++ {
			masked[i] ^= f.mask[i%4]
		}

		frame = append(frame, masked[:]...)
	}

	// fmt.Println(frame)
	if body != nil {
		frame = append(frame, body[:f.length]...)
	}

	_, err := conn.Write(frame)
	return err
}

func ReadWSFrame(reader *bufio.Reader) (*WSFrame, error) {
	f := &WSFrame{}

	b, err := reader.ReadByte()
	if err != nil {
		return f, err
	}
	f.final = (b & 0x80) != 0
	f.rsv1 = (b & 0x40) != 0
	f.rsv2 = (b & 0x20) != 0
	f.rsv3 = (b & 0x10) != 0
	f.opcode = Opcode(b & 0x0F)

	b, err = reader.ReadByte()
	if err != nil {
		return f, err
	}
	f.masked = (b & 0x80) != 0

	f.length = uint64(b & 0x7F)
	if f.length == 126 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = uint64(binary.BigEndian.Uint16(lenBuf))
	} else if f.length == 127 {
		lenBuf := make([]byte, 8)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = uint64(binary.BigEndian.Uint64(lenBuf))
	}

	f.payloadRemaining = f.length
	if f.masked {
		_, err := io.ReadFull(reader, f.mask[:])
		if err != nil {
			return f, err
		}
	}

	return f, nil
}
