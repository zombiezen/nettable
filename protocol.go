package nettable

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
)

var ErrKey = errors.New("Unrecognized ID")

const DefaultPort = 1735

const (
	codeString          = 0
	codeInt             = 1
	codeDouble          = 2
	codeTable           = 3
	codeTableAssignment = codeTable
	codeBoolFalse       = 4
	codeBoolTrue        = 5
	codeAssignment      = 6
	codeEmpty           = 7
	codeData            = 8
	codeOldData         = 9
	codeTransaction     = 10
	codeRemoval         = 11
	codeTableRequest    = 12
	codeID              = 1 << 7
	codeTableID         = 1 << 6
	codeConfirmation    = 1 << 5
	codePing            = codeConfirmation
	codeDenial          = 1 << 4
)

type encoder interface {
	encode(w io.Writer) error
}

// Client is a connection to a NetworkTables server.
type Client struct {
	rwc io.ReadWriteCloser
	r   *bufio.Reader

	lock         sync.Mutex
	nextTableID  ID
	nextKeyID    ID
	keys         map[ID]key    // local ID to key
	tables       map[ID]*Table // local ID to table
	tableNames   map[string]ID // string to local ID
	remoteKeys   map[ID]ID     // remote ID to local ID
	remoteTables map[ID]ID     // remote ID to local ID

	writeChan   chan encoder
	putRequests chan putRequest
	confirmChan chan bool

	wg sync.WaitGroup
}

// Dial returns a new client connected to the host on the default port.
func Dial(host string) (*Client, error) {
	c, err := net.Dial("tcp", host+":"+strconv.Itoa(DefaultPort))
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a new client from a connection.
func NewClient(rwc io.ReadWriteCloser) *Client {
	client := &Client{
		rwc:          rwc,
		r:            bufio.NewReaderSize(rwc, 1),
		keys:         make(map[ID]key),
		tables:       make(map[ID]*Table),
		tableNames:   make(map[string]ID),
		remoteKeys:   make(map[ID]ID),
		remoteTables: make(map[ID]ID),
		writeChan:    make(chan encoder),
		putRequests:  make(chan putRequest),
		confirmChan:  make(chan bool),
	}
	go client.read()
	client.wg.Add(2)
	go client.write()
	go client.requests()
	return client
}

// Close terminates the connection, closing the underlying connection.
func (c *Client) Close() error {
	close(c.writeChan)
	close(c.putRequests)
	c.wg.Wait()
	return c.rwc.Close()
}

// Table returns the table with the given name, requesting the table from the
// server if necessary.
func (c *Client) Table(name string) (*Table, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if id, ok := c.tableNames[name]; ok {
		return c.tables[id], nil
	}

	table := &Table{
		client: c,
		id:     c.nextTableID,
		values: make(map[string]Entry),
	}
	c.nextTableID++

	c.tableNames[name] = table.id
	c.tables[table.id] = table
	c.writeChan <- tableRequest{name, table.id}
	return table, nil
}

func (c *Client) read() {
	for {
		code, err := c.r.ReadByte()
		if err != nil {
			// TODO: log error
			break
		}
		c.r.UnreadByte()

		switch {
		case code >= codeID:
			err = c.readData()
		case code == codeTableAssignment:
			err = c.readTableAssignment()
		case code == codeAssignment:
			err = c.readAssignment()
		case code >= codeConfirmation:
			err = c.readConfirmation()
		case code >= codeDenial:
			err = c.readDenial()
		default:
			// TODO: log bad code
			c.r.ReadByte()
		}

		// TODO: log error
	}
}

func (c *Client) readData() error {
	remoteID, err := readID(c.r, codeID)
	if err != nil {
		return err
	}
	entry, err := decodeEntry(c.r)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	localID, ok := c.remoteKeys[remoteID]
	if !ok {
		// XXX: This isn't a key we know about. Ignore it.
		return nil
	}
	k := c.keys[localID]
	c.tables[k.TableID].update(k.Name, entry)
	c.writeChan <- confirmation(1)
	return nil
}

func (c *Client) readTableAssignment() error {
	// TODO: Check code byte?
	if _, err := c.r.ReadByte(); err != nil {
		return err
	}

	localTableID, err := readID(c.r, codeTableID)
	if err != nil {
		return err
	}
	remoteTableID, err := readID(c.r, codeTableID)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if _, ok := c.tables[localTableID]; !ok {
		// XXX: This isn't a table we know about. Ignore the assignment for now.
		return nil
	}

	c.remoteTables[remoteTableID] = localTableID
	return nil
}

func (c *Client) readAssignment() error {
	// TODO: Check code byte?
	if _, err := c.r.ReadByte(); err != nil {
		return err
	}
	remoteTableID, err := readID(c.r, codeTableID)
	if err != nil {
		return err
	}
	keyName, err := readString(c.r)
	if err != nil {
		return err
	}
	remoteID, err := readID(c.r, codeID)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	localTableID, ok := c.remoteTables[remoteTableID]
	if !ok {
		// XXX: This isn't a table we know about. Ignore the assignment for now.
		return nil
	}
	for _, k := range c.keys {
		if k.TableID == localTableID && k.Name == keyName {
			c.remoteKeys[remoteID] = k.ID
			return nil
		}
	}

	c.remoteKeys[remoteID] = c.getKeyID(localTableID, keyName)
	return nil
}

func (c *Client) readConfirmation() error {
	val, err := c.r.ReadByte()
	if err != nil {
		return err
	}

	if val < codeConfirmation || val >= codeConfirmation<<1 {
		return errors.New("readConfirmation called on bad message")
	}

	n := int(val &^ codeConfirmation)
	for i := 0; i < n; i++ {
		c.confirmChan <- true
	}
	return nil
}

func (c *Client) readDenial() error {
	val, err := c.r.ReadByte()
	if err != nil {
		return err
	}

	if val < codeDenial || val >= codeDenial<<1 {
		return errors.New("readDenial called on bad message")
	}

	n := int(val &^ codeDenial)
	for i := 0; i < n; i++ {
		c.confirmChan <- false
	}
	return nil
}

func (c *Client) write() {
	defer c.wg.Done()
	for e := range c.writeChan {
		// TODO: log error
		e.encode(c.rwc)
	}
}

func (c *Client) requests() {
	defer c.wg.Done()
requestLoop:
	for {
		select {
		case <-c.confirmChan:
			// TODO: log unnecessary confirm
		case req, ok := <-c.putRequests:
			if !ok {
				break requestLoop
			}

			c.sendPutRequest(req)
			success := <-c.confirmChan
			if success {
				req.Table.update(req.Key, req.Value)
				req.Result <- nil
			} else {
				req.Result <- ErrDenial
			}
		}
	}
}

func (c *Client) sendPutRequest(req putRequest) {
	c.lock.Lock()
	id := c.getKeyID(req.Table.id, req.Key)
	c.lock.Unlock()

	c.writeChan <- entryData{
		LocalID: id,
		Value:   req.Value,
	}
}

// getKeyID returns the local ID for the name.  If an ID does not exist, it is
// added.  This function requires the client to be locked.
func (c *Client) getKeyID(tableID ID, name string) ID {
	// TODO: This is slow. Try to have a key lookup later.
	for _, k := range c.keys {
		if k.TableID == tableID && k.Name == name {
			return k.ID
		}
	}

	k := key{TableID: tableID, Name: name, ID: c.nextKeyID}
	c.keys[k.ID] = k
	c.nextKeyID++
	c.writeChan <- k
	return k.ID
}

type ID uint32

type key struct {
	TableID ID
	ID      ID
	Name    string
}

func (k key) encode(w io.Writer) error {
	if _, err := w.Write([]byte{codeAssignment}); err != nil {
		return err
	}
	if err := writeID(w, codeTableID, k.TableID); err != nil {
		return err
	}
	if err := writeString(w, k.Name); err != nil {
		return err
	}
	if err := writeID(w, codeID, k.ID); err != nil {
		return err
	}
	return nil
}

type tableRequest struct {
	TableName string
	LocalID   ID
}

func (ta tableRequest) encode(w io.Writer) error {
	if _, err := w.Write([]byte{codeTableRequest}); err != nil {
		return err
	}
	if err := writeString(w, ta.TableName); err != nil {
		return err
	}
	if err := writeID(w, codeTableID, ta.LocalID); err != nil {
		return err
	}
	return nil
}

type putRequest struct {
	Table  *Table
	Key    string
	Value  Entry
	Result chan error
}

type entryData struct {
	LocalID ID
	Value   encoder
}

func (e entryData) encode(w io.Writer) error {
	if err := writeID(w, codeID, e.LocalID); err != nil {
		return err
	}
	return e.Value.encode(w)
}

type confirmation byte

func (c confirmation) encode(w io.Writer) error {
	_, err := w.Write([]byte{codeConfirmation | byte(c)})
	return err
}
