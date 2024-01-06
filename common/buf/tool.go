package buf

import (
	"io"
	"tiangong/common"
)

func WriteByte(buffer Buffer, data byte) error {
	return nil
}

func WriteInt(buffer Buffer, data int) error {
	return nil
}

func WriteBytes(buffer Buffer, reader io.Reader) (int, error) {
	return buffer.Write(reader)
}

func ReadByte(buffer Buffer) (byte, error) {
	one := [common.One]byte{}
	if n, err := buffer.Read(one[:]); err != nil || n != common.One {
		return 0, err
	}
	return one[0], nil
}

func ReadInt(buffer Buffer) (int, error) {
	return 0, nil
}