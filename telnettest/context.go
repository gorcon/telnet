package telnettest

import (
	"bufio"
	"net"
)

// Context represents the context of the current TELNET request.
type Context struct {
	server *Server
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	Auth   bool
}

// Server returns the Server instance.
func (c *Context) Server() *Server {
	return c.server
}

// Conn returns current TELNET Client connection.
func (c *Context) Conn() net.Conn {
	return c.conn
}

// Reader current TELNET Client connection Reader.
func (c *Context) Reader() *bufio.Reader {
	return c.reader
}

// Writer current TELNET Client connection Writer.
func (c *Context) Writer() *bufio.Writer {
	return c.writer
}
