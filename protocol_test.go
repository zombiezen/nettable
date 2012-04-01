package nettable

import (
	"bytes"
	"io"
	"testing"
)

func TestClientPut(t *testing.T) {
	buf := make([]byte, 64)
	conn := newMockConn()
	client := NewClient(conn)

	// Request table A
	table, err := client.Table("A")
	if err != nil {
		t.Errorf("Table() Error: %v", err)
	}
	buf = buf[:4]
	if _, err := io.ReadFull(conn.rServer, buf); err != nil {
		t.Errorf("Server read error: %v", err)
	}
	if !bytes.Equal(buf, []byte{0x0c, 0x01, 'A', 0x40}) {
		t.Errorf("Sent bad table request: %v", buf)
	}

	// Send table assignment from server
	conn.wServer.Write([]byte{0x03, 0x40, 0x4a})

	// Ensure A has no key B
	if table.Get("B") != nil {
		t.Errorf("Table starts off with a value for B")
	}

	// Put B
	ch := make(chan error)
	go func() {
		ch <- table.Put("B", Int(-27))
	}()

	// Server read assignment
	buf = buf[:5]
	if _, err := io.ReadFull(conn.rServer, buf); err != nil {
		t.Errorf("Server read error: %v", err)
	}
	if !bytes.Equal(buf, []byte{0x06, 0x40, 0x01, 'B', 0x80}) {
		t.Errorf("Sent bad assignment: %v", buf)
	}

	// Server read data
	buf = buf[:6]
	if _, err := io.ReadFull(conn.rServer, buf); err != nil {
		t.Errorf("Server read error: %v", err)
	}
	if !bytes.Equal(buf, []byte{0x80, 0x01, 0xff, 0xff, 0xff, 0xe5}) {
		t.Errorf("Sent bad data: %v", buf)
	}

	// Server write confirm
	conn.wServer.Write([]byte{0x21})

	// See what the client thinks
	err = <-ch
	if err != nil {
		t.Errorf("Put error: %v")
	}

	// Make sure put succeeded locally
	if val := table.Get("B"); val != Int(-27) {
		t.Errorf("After Get, key 'B' is %v, expected %v", val, Int(-27))
	}

	// Close client
	if err := client.Close(); err != nil {
		t.Errorf("Client close error: %v", err)
	}
}

type mockConn struct {
	rClient *io.PipeReader
	wClient *io.PipeWriter
	rServer *io.PipeReader
	wServer *io.PipeWriter
}

func newMockConn() *mockConn {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	return &mockConn{
		rClient: r1,
		wServer: w1,
		rServer: r2,
		wClient: w2,
	}
}

func (mc *mockConn) Read(p []byte) (int, error) {
	return mc.rClient.Read(p)
}

func (mc *mockConn) Write(p []byte) (int, error) {
	return mc.wClient.Write(p)
}

func (mc *mockConn) Close() error {
	mc.rClient.Close()
	mc.wClient.Close()
	return nil
}
