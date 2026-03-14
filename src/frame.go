package main

import (
	"bufio"
	"encoding/binary"
	"io"
)

type Frame interface {
	ReadHeader()
}

type WSFrame struct {
	reader *bufio.Reader
	final  bool
	opcode int
	masked bool
	length int64
	mask   int32
}

func NewWSFrame(reader *bufio.Reader) *WSFrame {
	return &WSFrame{
		reader: reader,
	}
}

func (f *WSFrame) ReadHeader() error {
	b, err := f.reader.ReadByte()
	if err != nil {
		return err
	}
	f.final = (b & 0x80) != 0
	f.opcode = int(b & 0x0F)

	b, err = f.reader.ReadByte()
	if err != nil {
		return err
	}
	f.masked = (b & 0x80) != 0

	f.length = int64(b & 0x7F)
	if f.length == 126 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(f.reader, lenBuf)
		if err != nil {
			return err
		}
		f.length = int64(binary.BigEndian.Uint16(lenBuf))
	}
	if f.length == 127 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(f.reader, lenBuf)
		if err != nil {
			return err
		}
		f.length = int64(binary.BigEndian.Uint64(lenBuf))
	}

	if f.masked {
		maskBuf := make([]byte, 4)
		_, err := io.ReadFull(f.reader, maskBuf)
		if err != nil {
			return err
		}
		f.mask = int32(binary.BigEndian.Uint32(maskBuf))
	}

	return nil
}
