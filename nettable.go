// Package nettable provides a client for the WPILib NetworkTables protocol.
package nettable

import (
	"errors"
	"sync"
)

var ErrDenial = errors.New("Value changed by peer")

// Table is a network-synchronized key-value collection.  You can obtain a table
// using a client.
type Table struct {
	client *Client
	id     ID

	values     map[string]Entry
	valuesLock sync.RWMutex
}

// Client returns the client this table is attached to.
func (t *Table) Client() *Client {
	return t.client
}

// Get returns the value for a given key, or nil if the key does not exist.
func (t *Table) Get(key string) Entry {
	t.valuesLock.RLock()
	defer t.valuesLock.RUnlock()
	return t.values[key]
}

// Put sends a value for the given key and returns any error encountered.
// ErrDenial is returned if the value changed on the server during the put.
func (t *Table) Put(key string, value Entry) error {
	ch := make(chan error)
	t.client.putRequests <- putRequest{t, key, value, ch}
	return <-ch
}

// update directly changes a value in the table.
func (t *Table) update(key string, value Entry) {
	t.valuesLock.Lock()
	defer t.valuesLock.Unlock()
	t.values[key] = value
}

/*
// Transaction batches table changes made in f into a single transaction.
func (t *Table) Transaction(f func()) {
	// code...
}
*/
