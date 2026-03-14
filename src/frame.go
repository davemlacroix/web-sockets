package main

import (
	"bufio"
	"encoding/binary"
	"io"
)

type Opcode int

const (
	Text   Opcode = 1
	Binary        = 2
	Close         = 8
)

type WSFrame struct {
	final  bool
	opcode Opcode
	masked bool
	length int64
	mask   int32
}

func NewWSFrame() *WSFrame {
	return &WSFrame{}
}
func NextWSFrame(reader *bufio.Reader) (*WSFrame, error) {
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

	f.length = int64(b & 0x7F)
	if f.length == 126 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = int64(binary.BigEndian.Uint16(lenBuf))
	}
	if f.length == 127 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = int64(binary.BigEndian.Uint64(lenBuf))
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
