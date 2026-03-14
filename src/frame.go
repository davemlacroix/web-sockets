package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

type Opcode int

const (
	Text   Opcode = 1
	Binary        = 2
	Close         = 8
)

type Frame interface {
	Type() Opcode
	ReadText() (string, error)
}

type WSFrame struct {
	reader *bufio.Reader
	final  bool
	opcode Opcode
	masked bool
	length int64
	mask   int32
}

func (f *WSFrame) Type() Opcode {
	return f.opcode
}

func (f *WSFrame) ReadText() (string, error) {
	if f.opcode != 1 {
		return "", errors.New("invalid frame type")
	}
	buf := make([]byte, f.length)
	_, err := io.ReadFull(f.reader, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func NextWSFrame(reader *bufio.Reader) (*WSFrame, error) {
	f := &WSFrame{
		reader: reader,
	}

	b, err := f.reader.ReadByte()
	if err != nil {
		return f, err
	}
	f.final = (b & 0x80) != 0
	f.opcode = Opcode(b & 0x0F)

	b, err = f.reader.ReadByte()
	if err != nil {
		return f, err
	}
	f.masked = (b & 0x80) != 0

	f.length = int64(b & 0x7F)
	if f.length == 126 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(f.reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = int64(binary.BigEndian.Uint16(lenBuf))
	}
	if f.length == 127 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(f.reader, lenBuf)
		if err != nil {
			return f, err
		}
		f.length = int64(binary.BigEndian.Uint64(lenBuf))
	}

	if f.masked {
		maskBuf := make([]byte, 4)
		_, err := io.ReadFull(f.reader, maskBuf)
		if err != nil {
			return f, err
		}
		f.mask = int32(binary.BigEndian.Uint32(maskBuf))
	}

	return f, nil
}
