package telnettest_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gorcon/telnet"
	"github.com/gorcon/telnet/telnettest"
)

func TestNewServer(t *testing.T) {
	handlers := func(c *telnettest.Context) {
		switch c.Request() {
		case "Can I help you?":
			c.Writer().WriteString(fmt.Sprintf("2020-12-07T21:37:00 31123.521 "+telnet.ResponseINFLayout, c.Request(), c.Conn().RemoteAddr()) + telnet.CRLF)
			c.Writer().WriteString("Can I help you?" + telnet.CRLF)
		case "What do you do?":
			c.Writer().WriteString(fmt.Sprintf("2020-12-07T21:37:00 31123.521 "+telnet.ResponseINFLayout, c.Request(), c.Conn().RemoteAddr()) + telnet.CRLF)
			c.Writer().WriteString("I do it all." + telnet.CRLF)
		default:
			c.Writer().WriteString(fmt.Sprintf("*** ERROR: unknown command '%s'", c.Request()) + telnet.CRLF)
		}

		c.Writer().Flush()
	}

	t.Run("with options", func(t *testing.T) {
		server := telnettest.NewServer(
			telnettest.SetSettings(telnettest.Settings{Password: "password"}),
			telnettest.SetCommandHandler(handlers),
		)
		defer server.Close()

		client, err := telnet.Dial(server.Addr(), "password", telnet.SetClearResponse(true))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		response, err := client.Execute("Can I help you?")
		if err != nil {
			t.Fatal(err)
		}

		if response != "Can I help you?" {
			t.Errorf("got %q, want \"Can I help you?\"", response)
		}
	})

	t.Run("unstarted", func(t *testing.T) {
		server := telnettest.NewUnstartedServer()
		server.Settings.Password = "password"
		server.Settings.AuthResponseDelay = 10 * time.Millisecond
		server.Settings.CommandResponseDelay = 10 * time.Millisecond
		server.SetCommandHandler(handlers)
		server.Start()
		defer server.Close()

		client, err := telnet.Dial(server.Addr(), "password", telnet.SetClearResponse(true))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		response, err := client.Execute("What do you do?")
		if err != nil {
			t.Fatal(err)
		}

		if response != "I do it all." {
			t.Errorf("got %q, want \"I do it all.\"", response)
		}
	})

	t.Run("authentication failed", func(t *testing.T) {
		server := telnettest.NewServer(telnettest.SetSettings(telnettest.Settings{Password: "password"}))
		defer server.Close()

		client, err := telnet.Dial(server.Addr(), "wrong")
		if err == nil {
			defer client.Close()
		}
		if !errors.Is(err, telnet.ErrAuthFailed) {
			t.Errorf("got error %v, want %v", err, telnet.ErrAuthFailed)
		}
	})

	t.Run("empty handler", func(t *testing.T) {
		server := telnettest.NewServer()
		defer server.Close()

		client, err := telnet.Dial(server.Addr(), "")
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		response, err := client.Execute("whatever")
		if err != nil {
			t.Fatal(err)
		}

		if response != "" {
			t.Errorf("got %q, want empty string", response)
		}
	})
}
