package telnet

import "time"

// Settings contains option to Conn.
type Settings struct {
	dialTimeout time.Duration
	exitCommand string
}

// DefaultSettings provides default deadline settings to Conn.
var DefaultSettings = Settings{
	dialTimeout: DefaultDialTimeout,
	exitCommand: DefaultExitCommand,
}

// Option allows to inject settings to Settings.
type Option func(s *Settings)

// SetDialTimeout injects dial Timeout to Settings.
func SetDialTimeout(timeout time.Duration) Option {
	return func(s *Settings) {
		s.dialTimeout = timeout
	}
}

// SetExitCommand injects telnet exit command.
func SetExitCommand(command string) Option {
	return func(s *Settings) {
		s.exitCommand = command
	}
}
