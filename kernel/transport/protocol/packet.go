package protocol

import (
	"io"
	"strconv"
	"tiangong/common/buf"
	"tiangong/common/errors"
	"unsafe"
)

const (
	PacketHeaderLen = int(unsafe.Sizeof((*PacketHeader)(nil)))
)

type PacketHeader struct {
	Len      uint16
	Rid      uint32
	Protocol byte
}

func (h *PacketHeader) Unmarshal(buffer buf.Buffer) error {
	if buffer.Len() < PacketHeaderLen {
		return errors.NewError("header([]byte) len too short, Minimum requirement "+strconv.Itoa(PacketHeaderLen)+"bytes", io.EOF)
	}
	h.Len, _ = buf.ReadUint16(buffer)
	h.Rid, _ = buf.ReadUint32(buffer)
	h.Protocol, _ = buf.ReadByte(buffer)
	return nil
}