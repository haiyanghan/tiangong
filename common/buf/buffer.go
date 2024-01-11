package buf

import (
	"io"
	"tiangong/common/errors"
)

const (
	_4K = 4096
)

var (
	NoSpace = errors.NewError("Unable to write, no space", nil)
)

// 4k bytes memory block
type block [_4K]byte

type Buffer interface {
	// Read byte arrays from the buffer, Returns the actual length read
	Read([]byte) (int, error)
	// Write a specified number of bytes to the buffer, Returns the actual length written
	Write(reader io.Reader, len int) (int, error)
	// Len Return remaining readable length
	Len() int
	// Release the Buffer
	Release()
	// Clear and reuse the Buffer
	Clear() error
}

func NewRingBuffer() Buffer {
	return &RingBuffer{
		len:    _4K,
		buffer: &block{},
	}
}

func WrapNew(bytes []byte) Buffer {
	return &ByteBuffer{
		bytes: bytes,
		len:   len(bytes),
	}
}

func Wrap(bytes []byte) Buffer {
	size := len(bytes)
	return &ByteBuffer{
		bytes: bytes,
		end:   size,
		len:   size,
	}
}
