package telnet

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDial(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("connection refused", func(t *testing.T) {
		conn, err := Dial("127.0.0.2:12345", MockPassword)
		if !assert.Error(t, err) {
			// Close connection if established.
			assert.NoError(t, conn.Close())
		}

		assert.EqualError(t, err, "dial tcp 127.0.0.2:12345: connect: connection refused")
	})

	t.Run("authentication failed", func(t *testing.T) {
		conn, err := Dial(server.Addr(), "wrong")
		if !assert.Error(t, err) {
			assert.NoError(t, conn.Close())
		}

		assert.EqualError(t, err, "authentication failed")
	})

	t.Run("auth success", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword, SetDialTimeout(5*time.Second))
		if assert.NoError(t, err) {
			assert.NoError(t, conn.Close())
		}
	})
}

func TestConn_Execute(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("incorrect command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer assert.NoError(t, conn.Close())

		result, err := conn.Execute("")
		assert.Equal(t, err, ErrCommandEmpty)
		assert.Equal(t, 0, len(result))

		result, err = conn.Execute(string(make([]byte, 1001)))
		assert.Equal(t, err, ErrCommandTooLong)
		assert.Equal(t, 0, len(result))
	})

	t.Run("closed network connection", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		assert.NoError(t, conn.Close())

		result, err := conn.Execute(MockCommandHelp)
		assert.EqualError(t, err, fmt.Sprintf("write tcp %s->%s: use of closed network connection", conn.LocalAddr(), conn.RemoteAddr()))
		assert.Equal(t, 0, len(result))
	})

	t.Run("unknown command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, conn.Close())
		}()

		result, err := conn.Execute("random")
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, "*** ERROR: unknown command 'random'", CRLF), result)
	})

	t.Run("success help command", func(t *testing.T) {
		conn, err := Dial(server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, conn.Close())
		}()

		result, err := conn.Execute(MockCommandHelp)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, MockCommandHelpResponse, CRLF), result)
	})
}

func TestConn_Interactive(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, server.Close())
		close(server.errors)
		for err := range server.errors {
			assert.NoError(t, err)
		}
	}()

	t.Run("unknown command", func(t *testing.T) {
		var r bytes.Buffer
		w := bytes.Buffer{}

		r.WriteString("random" + "\n")
		r.WriteString(ForcedExitCommand + "\n")

		err := DialInteractive(&r, &w, server.Addr(), MockPassword)
		if !assert.NoError(t, err) {
			return
		}

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, "*** ERROR: unknown command 'random'", CRLF), w.String())
	})

	t.Run("success help command", func(t *testing.T) {
		var r bytes.Buffer
		w := bytes.Buffer{}

		r.WriteString(MockCommandHelp + "\n")
		r.WriteString(ForcedExitCommand + "\n")

		err := DialInteractive(&r, &w, server.Addr(), MockPassword, SetExitCommand("exit"))
		if !assert.NoError(t, err) {
			return
		}

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Please enter password:%s%s%s%s%s", CRLF, AuthSuccess, CRLF, MockCommandHelpResponse, CRLF), w.String())
	})
}
