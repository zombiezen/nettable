package nettable

import (
	"bytes"
	"testing"
)

var idTests = []struct {
	Mask byte
	ID   ID
	Data []byte
}{
	{codeID, 0x00000000, []byte{0x80}},
	{codeID, 0x00000001, []byte{0x81}},
	{codeID, 0x0000007b, []byte{0xfb}},
	{codeID, 0x0000007c, []byte{0xfc, 0x7c}},
	{codeID, 0x000000ff, []byte{0xfc, 0xff}},
	{codeID, 0x00000100, []byte{0xfd, 0x01, 0x00}},
	{codeID, 0x0000ffff, []byte{0xfd, 0xff, 0xff}},
	{codeID, 0x00010000, []byte{0xfe, 0x01, 0x00, 0x00}},
	{codeID, 0x00ffffff, []byte{0xfe, 0xff, 0xff, 0xff}},
	{codeID, 0x01000000, []byte{0xff, 0x01, 0x00, 0x00, 0x00}},
	{codeID, 0xffffffff, []byte{0xff, 0xff, 0xff, 0xff, 0xff}},
	{codeTableID, 0x00000000, []byte{0x40}},
	{codeTableID, 0x00000001, []byte{0x41}},
	{codeTableID, 0x0000003b, []byte{0x7b}},
	{codeTableID, 0x0000003c, []byte{0x7c, 0x3c}},
	{codeTableID, 0x000000ff, []byte{0x7c, 0xff}},
	{codeTableID, 0x00000100, []byte{0x7d, 0x01, 0x00}},
	{codeTableID, 0x0000ffff, []byte{0x7d, 0xff, 0xff}},
	{codeTableID, 0x00010000, []byte{0x7e, 0x01, 0x00, 0x00}},
	{codeTableID, 0x00ffffff, []byte{0x7e, 0xff, 0xff, 0xff}},
	{codeTableID, 0x01000000, []byte{0x7f, 0x01, 0x00, 0x00, 0x00}},
	{codeTableID, 0xffffffff, []byte{0x7f, 0xff, 0xff, 0xff, 0xff}},
}

func TestReadID(t *testing.T) {
	var buf bytes.Buffer
	for i := range idTests {
		buf.Reset()
		buf.Write(idTests[i].Data)
		result, err := readID(&buf, idTests[i].Mask)
		if err != nil {
			t.Errorf("[%d] error: %v", i, err)
		} else if result != idTests[i].ID {
			t.Errorf("[%d]: readID(%v, %#02x) != %#08x (got %#08x)", i, idTests[i].Data, idTests[i].Mask, idTests[i].ID, result)
		}
	}
}

func TestWriteID(t *testing.T) {
	var buf bytes.Buffer
	for i := range idTests {
		buf.Reset()
		err := writeID(&buf, idTests[i].Mask, idTests[i].ID)
		if err != nil {
			t.Errorf("[%d] error: %v", i, err)
		} else if !bytes.Equal(buf.Bytes(), idTests[i].Data) {
			t.Errorf("[%d]: writeID(w, %#02x, %#08x) != %v (got %v)", i, idTests[i].Mask, idTests[i].ID, idTests[i].Data, buf.Bytes())
		}
	}
}

var stringTests = []struct {
	String string
	Data   []byte
}{
	{"", []byte("\x00")},
	{"Hello, World!", []byte("\x0dHello, World!")},
	{
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		[]byte("\xffAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\x00"),
	},
}

func TestReadString(t *testing.T) {
	var buf bytes.Buffer
	for i := range stringTests {
		buf.Reset()
		buf.Write(stringTests[i].Data)
		result, err := readString(&buf)
		if err != nil {
			t.Errorf("[%d] error: %v", i, err)
		} else if result != stringTests[i].String {
			t.Errorf("[%d]: readString(%v) != %q (got %q)", i, stringTests[i].Data, stringTests[i].String, result)
		}
	}
}

func TestWriteString(t *testing.T) {
	var buf bytes.Buffer
	for i := range stringTests {
		buf.Reset()
		err := writeString(&buf, stringTests[i].String)
		if err != nil {
			t.Errorf("[%d] error: %v", i, err)
		} else if !bytes.Equal(buf.Bytes(), stringTests[i].Data) {
			t.Errorf("[%d]: writeString(w, %q) != %v (got %v)", i, stringTests[i].String, stringTests[i].Data, buf.Bytes())
		}
	}
}
