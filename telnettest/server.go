// Package telnettest contains TELNET server for TELNET client testing.
// WIP: telnettest is not finally implemented. DO NOT USE IN PRODUCTION!
package telnettest

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorcon/telnet"
)

// AuthSuccessWelcomeMessage contains welcome TELNET Server message.
const AuthSuccessWelcomeMessage = `*** Connected with 7DTD server.
*** Server version: Alpha 18.4 (b4) Compatibility Version: Alpha 18.4
*** Dedicated server only build

Server IP:   127.0.0.1
Server port: 26900
Max players: 8
Game mode:   GameModeSurvival
World:       Navezgane
Game name:   My Game
Difficulty:  2

Press 'help' to get a list of all commands. Press 'exit' to end session.`

// Server is an TELNET server listening on a system-chosen port on the
// local loopback interface, for use in end-to-end TELNET tests.
type Server struct {
	Settings       Settings
	Listener       net.Listener
	addr           string
	authHandler    HandlerFunc
	commandHandler HandlerFunc
	connections    map[net.Conn]struct{}
	quit           chan bool
	wg             sync.WaitGroup
	mu             sync.Mutex
	closed         bool
}

// Settings contains configuration for TELNET Server.
type Settings struct {
	Password             string
	AuthResponseDelay    time.Duration
	CommandResponseDelay time.Duration
}

// HandlerFunc defines a function to serve TELNET requests.
type HandlerFunc func(c *Context)

// AuthHandler checks authorisation data and sets true if received password is valid.
func AuthHandler(c *Context) {
	const limit = 10

	_, _ = c.writer.WriteString(telnet.ResponseEnterPassword + telnet.CRLF)
	defer c.writer.Flush()

	for attempt := 1; attempt < limit; attempt++ {
		c.writer.Flush()

		p := make([]byte, len([]byte(c.server.Settings.Password)))
		_, _ = c.reader.Read(p)
		password := string(p)

		switch password {
		case c.server.Settings.Password:
			_, _ = c.writer.WriteString(telnet.ResponseAuthSuccess + telnet.CRLF + telnet.CRLF + telnet.CRLF + telnet.CRLF)
			_, _ = c.writer.WriteString(AuthSuccessWelcomeMessage + telnet.CRLF + telnet.CRLF)

			c.Auth = true

			return
		case "unexpect":
			_, _ = c.writer.WriteString("My spoon is too big" + telnet.CRLF + telnet.CRLF)

			return
		default:
			_, _ = c.writer.WriteString(telnet.ResponseAuthIncorrectPassword + telnet.CRLF)
		}
	}

	_, _ = c.writer.WriteString(telnet.ResponseAuthTooManyFails + telnet.CRLF)
}

// EmptyHandler responses with empty body.
func EmptyHandler(c *Context) {
	scanner := bufio.NewScanner(c.reader)

	for {
		scanned := scanner.Scan()
		if !scanned {
			if err := scanner.Err(); err != nil {
				if err != io.EOF {
					panic(fmt.Errorf("handle read request error: %w", err))
				}

				return
			}

			break
		}

		request := scanner.Text()

		switch request {
		case "":
		case "help":
			_, _ = c.writer.WriteString(fmt.Sprintf("2020-11-14T23:09:20 31220.643 "+telnet.ResponseINFLayout, request, c.conn.RemoteAddr()) + telnet.CRLF) //nolint:lll // OMG
			_, _ = c.writer.WriteString("lorem ipsum dolor sit amet" + telnet.CRLF)
		case "exit":
		default:
			_, _ = c.writer.WriteString(fmt.Sprintf("*** ERROR: unknown command '%s'", request) + telnet.CRLF)
		}

		c.writer.Flush()
	}
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("telnettest: failed to listen on a port: %v", err))
	}

	return l
}

// NewServer returns a running TELNET Server or nil if an error occurred.
// The caller should call Close when finished, to shut it down.
func NewServer(options ...Option) *Server {
	server := NewUnstartedServer(options...)
	server.Start()

	return server
}

// NewUnstartedServer returns a new Server but doesn't start it.
// After changing its configuration, the caller should call Start.
// The caller should call Close when finished, to shut it down.
func NewUnstartedServer(options ...Option) *Server {
	server := Server{
		Listener:       newLocalListener(),
		authHandler:    AuthHandler,
		commandHandler: EmptyHandler,
		connections:    make(map[net.Conn]struct{}),
		quit:           make(chan bool),
	}

	for _, option := range options {
		option(&server)
	}

	return &server
}

// SetAuthHandler injects HandlerFunc with authorisation data checking.
func (s *Server) SetAuthHandler(handler HandlerFunc) {
	s.authHandler = handler
}

// SetCommandHandler injects HandlerFunc with commands processing.
func (s *Server) SetCommandHandler(handler HandlerFunc) {
	s.commandHandler = handler
}

// Start starts a server from NewUnstartedServer.
func (s *Server) Start() {
	if s.addr != "" {
		panic("server already started")
	}

	s.addr = s.Listener.Addr().String()
	s.goServe()
}

// Close shuts down the Server.
func (s *Server) Close() {
	if s.closed {
		return
	}

	s.closed = true
	close(s.quit)
	s.Listener.Close()

	// Waiting for server connections.
	s.wg.Wait()

	s.mu.Lock()
	for c := range s.connections {
		// Force-close any connections.
		s.closeConn(c)
	}
	s.mu.Unlock()
}

// Addr returns IPv4 string Server address.
func (s *Server) Addr() string {
	return s.addr
}

// NewContext returns a Context instance.
func (s *Server) NewContext(conn net.Conn) *Context {
	return &Context{server: s, conn: conn, reader: bufio.NewReader(conn), writer: bufio.NewWriter(conn)}
}

// serve handles incoming requests until a stop signal is given with Close.
func (s *Server) serve() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.isRunning() {
				panic(fmt.Errorf("serve error: %w", err))
			}

			return
		}

		s.wg.Add(1)

		go s.handle(conn)
	}
}

// serve calls serve in goroutine.
func (s *Server) goServe() {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		s.serve()
	}()
}

// handle handles incoming client conn.
func (s *Server) handle(conn net.Conn) {
	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.closeConn(conn)
		s.wg.Done()
	}()

	ctx := s.NewContext(conn)

	defer ctx.writer.Flush()

	s.authHandler(ctx)

	if !ctx.Auth {
		return
	}

	s.commandHandler(ctx)
}

// isRunning returns true if Server is running and false if is not.
func (s *Server) isRunning() bool {
	select {
	case <-s.quit:
		return false
	default:
		return true
	}
}

// closeConn closes a client conn and removes it from connections map.
func (s *Server) closeConn(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := conn.Close(); err != nil {
		panic(fmt.Errorf("close conn error: %w", err))
	}

	delete(s.connections, conn)
}
