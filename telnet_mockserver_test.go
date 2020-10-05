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

	return err
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

	w.WriteString("Please enter password:" + CRLF)
	defer w.Flush()

	for attempt := 1; attempt <= limit; attempt++ {
		w.Flush()

		p := make([]byte, len([]byte(MockPassword)))
		r.Read(p)
		password := string(p)

		switch password {
		case MockPassword:
			w.WriteString(AuthSuccess + CRLF)
			return true
		default:
			if attempt == limit {
				w.WriteString(AuthTooManyFails + CRLF)
				return false
			}

			w.WriteString(AuthIncorrectPassword + CRLF)
		}
	}

	return false
}
