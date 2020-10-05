package telnet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// MaxCommandLen is an artificial restriction, but it will help in case of
// random large queries.
const MaxCommandLen = 1000

// DefaultDialTimeout provides default auth timeout to remote server.
const DefaultDialTimeout = 5 * time.Second

// DefaultExitCommand provides default TELNET exit command.
const DefaultExitCommand = "exit"

// ForcedExitCommand provides forced TELNET exit command.
const ForcedExitCommand = ":q"

// CRLF moves the cursor to the next line and then moves it to the beginning.
const CRLF = "\r\n"

// ReceiveWaitPeriod is a delay to receive data from the server.
const ReceiveWaitPeriod = 3 * time.Millisecond

// ExecuteTickTimeout is execute read timeout.
const ExecuteTickTimeout = 1 * time.Second

// Remote server response messages.
const (
	AuthSuccess           = "Logon successful."
	AuthIncorrectPassword = "Password incorrect, please enter password:"
	AuthTooManyFails      = "Too many failed login attempts!"
)

var (
	// ErrAuthFailed is returned when 7 Days to Die server rejected
	// sent password.
	ErrAuthFailed = errors.New("authentication failed")

	// ErrCommandTooLong is returned when executed command length is bigger
	// than MaxCommandLen characters.
	ErrCommandTooLong = errors.New("command too long")

	// ErrCommandEmpty is returned when executed command length equal 0.
	ErrCommandEmpty = errors.New("command too small")

	// ErrMultiErrorOccurred is returned when close connection failed with
	// error after auth failed.
	ErrMultiErrorOccurred = errors.New("an error occurred while handling another error")
)

// Conn is TELNET connection.
type Conn struct {
	conn     net.Conn
	settings Settings
	reader   io.Reader
	writer   io.Writer

	w bytes.Buffer
}

// Dial creates a new authorized TELNET connection.
func Dial(address string, password string, options ...Option) (*Conn, error) {
	settings := DefaultSettings

	for _, option := range options {
		option(&settings)
	}

	conn, err := net.DialTimeout("tcp", address, settings.dialTimeout)
	if err != nil {
		// Failed to open TCP conn to the server.
		return nil, err
	}

	client := Conn{conn: conn, settings: settings, reader: conn, writer: conn}

	// TODO: Graceful close.
	go client.processReadResponse(&client.w)

	if err := client.auth(password); err != nil {
		// Failed to auth conn with the server.
		if err2 := client.Close(); err2 != nil {
			return &client, fmt.Errorf("%w: %v. Previous error: %v", ErrMultiErrorOccurred, err2, err)
		}

		return &client, err
	}

	return &client, nil
}

// DialInteractive parses commands from input reader, executes them on remote
// server and writes responses to output writer.
func DialInteractive(r io.Reader, w io.Writer, address string, password string, options ...Option) error {
	settings := DefaultSettings

	for _, option := range options {
		option(&settings)
	}

	conn, err := net.DialTimeout("tcp", address, settings.dialTimeout)
	if err != nil {
		// Failed to open TCP conn to the server.
		return err
	}

	client := Conn{conn: conn, settings: settings, reader: conn, writer: conn}
	defer client.Close()

	if password != "" {
		if _, err := client.write([]byte(password + CRLF)); err != nil {
			return err
		}
	}

	// TODO: Graceful close.
	go client.processReadResponse(w)

	return client.interactive(r, w)
}

// Execute sends command string to execute to the remote TELNET server.
func (c *Conn) Execute(command string) (string, error) {
	response, err := c.execute(command)
	if err != nil {
		return response, err
	}

	_, err = c.write([]byte(c.settings.exitCommand + CRLF))

	time.Sleep(ReceiveWaitPeriod)

	return c.w.String(), err
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Close closes the client connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// auth authenticates client for the next requests.
func (c *Conn) auth(password string) error {
	response, err := c.execute(password)
	if err != nil {
		return err
	}

	if !strings.Contains(response, AuthSuccess+CRLF) {
		return ErrAuthFailed
	}

	return nil
}

// execute sends command string to execute to the remote TELNET server.
func (c *Conn) execute(command string) (string, error) {
	if command == "" {
		return "", ErrCommandEmpty
	}

	if len(command) > MaxCommandLen {
		return "", ErrCommandTooLong
	}

	if _, err := c.write([]byte(command + CRLF)); err != nil {
		return "", err
	}

	time.Sleep(ExecuteTickTimeout)

	return c.w.String(), nil
}

// interactive reads commands from reader in terminal mode and sends them
// to execute to the remote TELNET server.
func (c *Conn) interactive(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		command := scanner.Text()

		if command == ForcedExitCommand {
			command = c.settings.exitCommand
		}

		if _, err := c.write([]byte(command + CRLF)); err != nil {
			return err
		}

		if command == c.settings.exitCommand {
			break
		}
	}

	time.Sleep(ReceiveWaitPeriod)

	return nil
}

// write sends data to established TELNET connection.
func (c *Conn) write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

// read reads structured binary data from c.conn into byte array.
func (c *Conn) read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

// processReadResponse reads response data from TELNET connection
// and writes them to writer (Stdout).
func (c *Conn) processReadResponse(writer io.Writer) {
	var buffer [1]byte
	p := buffer[:]

	for {
		// Read 1 byte.
		n, err := c.read(p)
		if n <= 0 && err == nil {
			continue
		} else if n <= 0 && err != nil {
			break
		}

		_, _ = writer.Write(p)
	}
}