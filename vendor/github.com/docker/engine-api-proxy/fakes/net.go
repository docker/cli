package fakes

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/docker/engine-api-proxy/types"
)

// make sure interfaces are properly implemented
var (
	_ net.Listener                    = &FakeListener{}
	_ net.Addr                        = fakeAddr{}
	_ net.Addr                        = connAddr{}
	_ net.Conn                        = &Conn{}
	_ types.CloseWriter               = &Conn{}
	_ types.ReadWriteCloseCloseWriter = &Conn{}
)

// FakeListener allows for in-memory, in-process proxying.
// It is the link between the CLI's http client and the proxy server.
// How it works:
// - each in-memory/in-process proxy has a FakeListener
// - clients obtain connections by calling DialContext()
// - this creates two net.Conn, that are linked together
// - one is returned to the client, the other is pushed in a queue
// - server obtains connections from the queue by calling Accept()
type FakeListener struct {
	queue chan net.Conn
}

// NewListener creates a new FakeListener and returns a pointer to it
func NewListener() *FakeListener {
	return &FakeListener{queue: make(chan net.Conn)}
}

// Accept returns the next connection in the queue.
// This function is intended to be used by the in-memory proxy server.
func (f *FakeListener) Accept() (net.Conn, error) {
	select {
	case newConnection := <-f.queue:
		return newConnection, nil
	}
}

// Close ...
func (f *FakeListener) Close() error {
	return nil
}

// Addr ...
func (f *FakeListener) Addr() net.Addr {
	return fakeAddr{}
}

// DialContext returns client connection to the caller.
func (f *FakeListener) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	serverConn, clientConn := Pipe()
	f.queue <- serverConn
	return clientConn, nil
}

// Pipe creates and return to connected Conn structs
// (like the net.Pipe() function)
func Pipe() (*Conn, *Conn) {
	a1, a2 := net.Pipe()
	b1, b2 := net.Pipe()

	p1r := &pipe{
		reader: a1,
		writer: nil,
	}
	p1w := &pipe{
		reader: nil,
		writer: b1,
	}

	p2r := &pipe{
		reader: b2,
		writer: nil,
	}
	p2w := &pipe{
		reader: nil,
		writer: a2,
	}

	conn1 := &Conn{
		reader: p1r,
		writer: p1w,
	}
	conn2 := &Conn{
		reader: p2r,
		writer: p2w,
	}

	return conn1, conn2
}

// Types

type fakeAddr struct{}

func (a fakeAddr) Network() string {
	return "memory"
}

func (a fakeAddr) String() string {
	return "fake"
}

// Conn implements the net.Conn interface,
// but also the CloseWriter and CloseReader interfaces.
// That means it is possible to close each connection way (read or write) separately
type Conn struct {
	reader *pipe
	writer *pipe
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.writer.Write(b)
}

func (c *Conn) Close() error {
	err := c.reader.Close()
	err1 := c.writer.Close()
	if err == nil {
		err = err1
	}
	return err
}

func (c *Conn) LocalAddr() net.Addr {
	return connAddr{}
}

func (c *Conn) RemoteAddr() net.Addr {
	return connAddr{}
}

func (c *Conn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "fakeconn", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "fakeconn", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "fakeconn", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

// CloseWrite is the implementation of the CloseWriter interface
func (c *Conn) CloseWrite() error {
	return c.writer.Close()
}

// CloseRead is the implementation of the CloseReader interface
func (c *Conn) CloseRead() error {
	return c.reader.Close()
}

/////

type connAddr struct{}

func (connAddr) Network() string {
	return "fakeconn"
}

func (connAddr) String() string {
	return "fakeconn"
}

/////

type pipe struct {
	reader net.Conn
	writer net.Conn
}

func (p *pipe) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

func (p *pipe) Write(b []byte) (n int, err error) {
	return p.writer.Write(b)
}

func (p *pipe) Close() error {
	var err error
	if p.reader != nil {
		err = p.reader.Close()
	}
	var err1 error
	if p.writer != nil {
		err1 = p.writer.Close()
	}
	if err == nil {
		err = err1
	}
	return err
}
