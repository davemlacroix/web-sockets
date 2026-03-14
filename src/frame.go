package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
)

type Opcode int

const (
	Text   Opcode = 1
	Binary        = 2
	Close         = 8
)

type Frame interface {
	Write(conn net.Conn, content []byte) error
}

type WSFrame struct {
	final  bool
	opcode Opcode
	masked bool
	length uint64
	mask   int32
}

func NewWSFrame() *WSFrame {
	return &WSFrame{}
}

func (f *WSFrame) Write(conn net.Conn, content []byte) error {
	var header []byte
	b1 := byte(0)
	if f.final {
		b1 |= 0x80
	}
	b1 |= byte(f.opcode) & 0x0F
	header = append(header, b1)

	b2 := byte(0)
	if f.masked {
		b2 |= 0x80
	}

	if f.length <= 125 {
		b2 |= byte(f.length)
		header = append(header, b2)
	} else if f.length <= 0xFFFF {
		b2 |= 126
		header = append(header, b2)
		extLen := make([]byte, 2)
		binary.BigEndian.PutUint16(extLen, uint16(f.length))
		header = append(header, extLen...)
	} else {
		b2 |= 127
		header = append(header, b2)
		extLen := make([]byte, 8)
		binary.BigEndian.PutUint64(extLen, uint64(f.length))
		header = append(header, extLen...)
	}

	if f.masked {
		maskBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(maskBuf, uint32(f.mask))
		header = append(header, maskBuf...)
	}

	frame := append(header, content[:f.length]...)
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
	}
	if f.length == 127 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = uint64(binary.BigEndian.Uint64(lenBuf))
	}

	if f.masked {
		maskBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, maskBuf)
		if err != nil {
			return f, err
		}
		f.mask = int32(binary.BigEndian.Uint32(maskBuf))
	}

	return f, nil
}
