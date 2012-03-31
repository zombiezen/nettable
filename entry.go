package nettable

import (
	"encoding/binary"
	"errors"
	"io"
)

// An Entry can be sent or received as a value in a table.
type Entry interface {
	encoder
}

// Int is an integral entry.
type Int int32

func (i Int) encode(w io.Writer) error {
	if _, err := w.Write([]byte{codeInt}); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, i)
}

// Double is a floating-point entry.
type Double float64

func (n Double) encode(w io.Writer) error {
	if _, err := w.Write([]byte{codeDouble}); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, n)
}

// Bool is a boolean entry.
type Bool bool

func (b Bool) encode(w io.Writer) error {
	var data [1]byte
	if b {
		data[0] = codeBoolTrue
	} else {
		data[0] = codeBoolFalse
	}
	_, err := w.Write(data[:])
	return err
}

// decodeEntry reads a single entry from r and returns any error encountered.
func decodeEntry(r io.Reader) (Entry, error) {
	code, err := readByte(r)
	if err != nil {
		return nil, err
	}
	switch code {
	case codeBoolTrue:
		return Bool(true), nil
	case codeBoolFalse:
		return Bool(false), nil
	case codeInt:
		var i Int
		err = binary.Read(r, binary.BigEndian, &i)
		return i, err
	case codeDouble:
		var n Double
		err = binary.Read(r, binary.BigEndian, &n)
		return n, err
	}
	return nil, errors.New("Unrecognized entry code")
}
