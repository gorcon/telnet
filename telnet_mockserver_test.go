package telnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

const (
	MockAddress  = "127.0.0.1:0"
	MockPassword = "password"

	MockCommandHelp         = "help"
	MockCommandHelpResponse = "lorem ipsum dolor sit amet"
)

const MockAuthSuccessWelcomeMessage = `*** Connected with 7DTD server.
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

// MockServer is a mock Source TELNET protocol server.
type MockServer struct {
	addr        string
	listener    net.Listener
	connections map[net.Conn]struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	errors      chan error
	quit        chan bool
}

// NewMockServer returns a running MockServer or nil if an error occurred.
func NewMockServer() (*MockServer, error) {
	listener, err := net.Listen("tcp", MockAddress)
	if err != nil {
		return nil, err
	}

	server := &MockServer{
		listener:    listener,
		connections: make(map[net.Conn]struct{}),
		errors:      make(chan error, 10),
		quit:        make(chan bool),
	}
	server.addr = server.listener.Addr().String()

	server.wg.Add(1)
	go server.serve()

	return server, nil
}

func MustNewMockServer() *MockServer {
	server, err := NewMockServer()
	if err != nil {
		panic(err)
	}

	return server
}

// Close shuts down the MockServer.
func (s *MockServer) Close() error {
	close(s.quit)

	err := s.listener.Close()

	// Waiting for server connections.
	s.wg.Wait()

	// And close remaining connections.
	s.mu.Lock()
	for c := range s.connections {
		// Close connections and add original error if occurred.
		if err2 := c.Close(); err2 != nil {
			if err == nil {
				err = fmt.Errorf("close connenction error: %s", err2)
			} else {
				err = fmt.Errorf("close connenction error: %s. Previous error: %s", err2, err)
			}
		}
	}
	s.mu.Unlock()

	close(s.errors)

	return err
}

func (s *MockServer) MustClose() {
	if s == nil {
		panic("server is not running")
	}

	if err := s.Close(); err != nil {
		panic(err)
	}

	for err := range s.errors {
		panic(err)
	}
}

// Addr returns IPv4 string MockServer address.
func (s *MockServer) Addr() string {
	return s.addr
}

// serve handles incoming requests until a stop signal is given with Close.
func (s *MockServer) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.isRunning() {
				s.reportError(fmt.Errorf("serve error: %s", err))
			}

			return
		}

		s.wg.Add(1)
		go s.handle(conn)
	}
}

// handle handles incoming client conn.
func (s *MockServer) handle(conn net.Conn) {
	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.closeConnection(conn)
		s.wg.Done()
	}()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	defer w.Flush()

	if !s.auth(r, w) {
		return
	}

	scanner := bufio.NewScanner(r)
	for {
		scanned := scanner.Scan()
		if !scanned {
			if err := scanner.Err(); err != nil {
				if err == io.EOF {
					return
				}

				s.reportError(fmt.Errorf("handle read request error: %s", err))
				return
			}

			break
		}

		request := scanner.Text()

		switch request {
		case "":
		case MockCommandHelp:
			w.WriteString(fmt.Sprintf("2020-11-14T23:09:20 31220.643 "+ResponseINFLayout, request, conn.RemoteAddr()) + CRLF)
			w.WriteString(MockCommandHelpResponse + CRLF)
		case "exit":
		default:
			w.WriteString(fmt.Sprintf("*** ERROR: unknown command '%s'", request) + CRLF)
		}

		w.Flush()
	}
}

// isRunning returns true if MockServer is running and false if is not.
func (s *MockServer) isRunning() bool {
	select {
	case <-s.quit:
		return false
	default:
		return true
	}
}

// closeConnection closes a client conn and removes it from connections map.
func (s *MockServer) closeConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := conn.Close(); err != nil {
		s.reportError(fmt.Errorf("close conn error: %s", err))
	}
	delete(s.connections, conn)
}

// reportError writes error to errors channel.
func (s *MockServer) reportError(err error) bool {
	if err == nil {
		return false
	}

	select {
	case s.errors <- err:
		return true
	default:
		fmt.Printf("erros channel is locked: %s\n", err)
		// panic("erros channel is locked")
		return false
	}
}

// auth checks authorisation data and returns true if received password is valid.
func (s *MockServer) auth(r *bufio.Reader, w *bufio.Writer) bool {
	const limit = 10

	w.WriteString(ResponseEnterPassword + CRLF)
	defer w.Flush()

	for attempt := 1; attempt <= limit; attempt++ {
		w.Flush()

		p := make([]byte, len([]byte(MockPassword)))
		r.Read(p)
		password := string(p)

		switch password {
		case MockPassword:
			w.WriteString(ResponseAuthSuccess + CRLF + CRLF + CRLF + CRLF)
			w.WriteString(MockAuthSuccessWelcomeMessage + CRLF + CRLF)
			return true
		case "unexpect":
			w.WriteString("My spoon is too big" + CRLF + CRLF)
			return false
		default:
			if attempt == limit {
				w.WriteString(ResponseAuthTooManyFails + CRLF)
				return false
			}

			w.WriteString(ResponseAuthIncorrectPassword + CRLF)
		}
	}

	return false
}
