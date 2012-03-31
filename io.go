package nettable

import (
	"errors"
	"io"
	"strings"
)

// readID decodes an ID from r. mask should be either codeID or codeTableID.
func readID(r io.Reader, mask byte) (ID, error) {
	var buf [4]byte

	if _, err := io.ReadFull(r, buf[:1]); err != nil {
		return 0, err
	}

	buf[0] &^= mask
	if buf[0] < mask-4 {
		return ID(buf[0]), nil
	}

	// The bottom 3 bits are used to indicate how large the ID is if all other
	// bits above (up to the mask) are set high.
	n := (buf[0] & 0x03) + 1
	if _, err := io.ReadFull(r, buf[:n]); err != nil {
		return 0, err
	}
	id := ID(0)
	for _, b := range buf[:n] {
		id = id<<8 | ID(b)
	}
	return id, nil
}

// writeID encodes id to w. mask should be either codeID or codeTableID.
func writeID(w io.Writer, mask byte, id ID) error {
	var buf [5]byte

	if id < ID(mask-4) {
		buf[0] = mask | byte(id)
		_, err := w.Write(buf[:1])
		return err
	}

	var p []byte
	switch {
	case id < 1<<8:
		buf[1] = byte(id)
		p = buf[:2]
	case id < 1<<16:
		buf[1] = byte(id >> 8)
		buf[2] = byte(id)
		p = buf[:3]
	case id < 1<<24:
		buf[1] = byte(id >> 16)
		buf[2] = byte(id >> 8)
		buf[3] = byte(id)
		p = buf[:4]
	default:
		buf[1] = byte(id >> 24)
		buf[2] = byte(id >> 16)
		buf[3] = byte(id >> 8)
		buf[4] = byte(id)
		p = buf[:5]
	}
	buf[0] = mask | (mask-1)&^3 | byte(len(p)-2)
	_, err := w.Write(p)
	return err
}

const (
	beginStringByte = 0xff
	endStringByte   = 0x00
)

// readString decodes a string in NetworkTable format from r.
func readString(r io.Reader) (string, error) {
	lenByte, err := readByte(r)
	if err != nil {
		return "", err
	}
	if lenByte < beginStringByte {
		// Pascal-style length byte
		buf := make([]byte, lenByte)
		n, err := io.ReadFull(r, buf)
		return string(buf[:n]), err
	}

	// Null-terminated string
	buf := make([]byte, 0, 360)
	for {
		size := len(buf)
		if _, err = r.Read(buf[size : size+1]); err != nil {
			break
		}
		if buf[:size+1][size] == endStringByte {
			break
		} else {
			buf = buf[:size+1]
		}
	}
	return string(buf), err
}

// writeString encodes s to w in NetworkTable format.
func writeString(w io.Writer, s string) error {
	// String must not have nulls
	if strings.ContainsRune(s, 0) {
		return errors.New("A NetworkTable string cannot contain null bytes")
	}

	if len(s) < beginStringByte {
		if _, err := w.Write([]byte{byte(len(s))}); err != nil {
			return err
		}
		_, err := io.WriteString(w, s)
		return err
	}

	if _, err := w.Write([]byte{beginStringByte}); err != nil {
		return err
	}
	if _, err := io.WriteString(w, s); err != nil {
		return err
	}
	if _, err := w.Write([]byte{endStringByte}); err != nil {
		return err
	}
	return nil
}

// readByte reads a single byte from r.  If r implements ByteReader, then
// ReadByte is called.
func readByte(r io.Reader) (byte, error) {
	if br, ok := r.(io.ByteReader); ok {
		return br.ReadByte()
	}
	var buf [1]byte
	_, err := io.ReadFull(r, buf[:])
	return buf[0], err
}
