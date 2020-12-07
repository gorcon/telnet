package telnettest_test

import (
	"fmt"
	"log"
	"time"

	"github.com/gorcon/telnet"
	"github.com/gorcon/telnet/telnettest"
)

func ExampleServer() {
	server := telnettest.NewServer(
		telnettest.SetSettings(telnettest.Settings{Password: "password"}),
		telnettest.SetAuthHandler(func(c *telnettest.Context) {
			switch c.Request() {
			case c.Server().Settings.Password:
				c.Writer().WriteString(telnet.ResponseAuthSuccess + telnet.CRLF + telnet.CRLF + telnet.CRLF + telnet.CRLF)
				c.Writer().WriteString(telnettest.AuthSuccessWelcomeMessage + telnet.CRLF + telnet.CRLF)

				c.Auth.Success = true
				c.Auth.Break = true
			default:
				c.Writer().WriteString(telnet.ResponseAuthIncorrectPassword + telnet.CRLF)
			}
		}),
		telnettest.SetCommandHandler(func(c *telnettest.Context) {
			switch c.Request() {
			case "Hello, server":
				c.Writer().WriteString(fmt.Sprintf(time.Now().Format("2006-01-02T15:04:05 00000.000 ")+telnet.ResponseINFLayout, c.Request(), c.Conn().RemoteAddr()) + telnet.CRLF)
				c.Writer().WriteString("Hello, client" + telnet.CRLF)
			default:
				c.Writer().WriteString(fmt.Sprintf("*** ERROR: unknown command '%s'", c.Request()) + telnet.CRLF)
			}

			c.Writer().Flush()
		}),
	)
	defer server.Close()

	client, err := telnet.Dial(server.Addr(), "password", telnet.SetClearResponse(true))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	response, err := client.Execute("Hello, server")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)

	response, err = client.Execute("Hi!")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)

	// Output:
	// Hello, client
	// *** ERROR: unknown command 'Hi!'
}
