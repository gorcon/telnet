# telnet

[![Go Report Card](https://goreportcard.com/badge/github.com/gorcon/telnet)](https://goreportcard.com/report/github.com/gorcon/telnet)
[![Build Status](https://travis-ci.org/gorcon/telnet.svg?branch=master)](https://travis-ci.org/gorcon/telnet)
[![Coverage](https://gocover.io/_badge/github.com/gorcon/telnet?0 "coverage")](https://gocover.io/github.com/gorcon/telnet)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/gorcon/telnet)

7 Days to Die remote access to game [Command Console](https://7daystodie.gamepedia.com/Command_Console). This is not full [TELNET](https://en.wikipedia.org/wiki/Telnet) protocol implementation.

## Supported Games

* [7 Days to Die](https://store.steampowered.com/app/251570) 

Open pull request if you have successfully used a package with another game with telnet support and add it to the list.

## Install

```text
go get github.com/gorcon/telnet
```

See [Changelog](CHANGELOG.md) for release details.

## Usage

### Execute single command

```go
package main

import (
	"log"
	"fmt"

	"github.com/gorcon/telnet"
)

func main() {
	conn, err := telnet.Dial("172.19.0.2:8081", "banana")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	response, err := conn.Execute("help")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println(response)	
}
```

### Interactive CLI mode

```go
package main

import (
	"log"
	"os"

	"github.com/gorcon/telnet"
)

func main() {
	err := telnet.DialInteractive(os.Stdin, os.Stdout, "172.19.0.2:8081", "")
	if err != nil {
		log.Println(err)
	}
}
```

## Requirements

Go 1.15 or higher

## Contribute

Contributions are more than welcome! 

If you think that you have found a bug, create an issue and publish the minimum amount of code triggering the bug so 
it can be reproduced.

If you want to fix the bug then you can create a pull request. If possible, write a test that will cover this bug.

## License

MIT License, see [LICENSE](LICENSE)
