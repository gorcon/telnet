package telnettest

import (
	"bufio"
	"net"
)

// Context represents the context of the current TELNET request.
type Context struct {
	Auth struct {
		Success bool
		Break   bool
	}
	server  *Server
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	request string
}

// Server returns the Server instance.
func (c *Context) Server() *Server {
	return c.server
}

// Conn returns current TELNET Client connection.
func (c *Context) Conn() net.Conn {
	return c.conn
}

// Reader returns current TELNET Client connection Reader.
func (c *Context) Reader() *bufio.Reader {
	return c.reader
}

// Writer returns current TELNET Client connection Writer.
func (c *Context) Writer() *bufio.Writer {
	return c.writer
}

// Request returns current request body string.
func (c *Context) Request() string {
	return c.request
}
