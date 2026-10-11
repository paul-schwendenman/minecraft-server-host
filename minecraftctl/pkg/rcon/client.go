package rcon

import (
	"errors"
	"fmt"

	"github.com/gorcon/rcon"
	"github.com/paul/minecraftctl/pkg/config"
)

// ErrAuthFailed means the server rejected the password; retrying won't help.
// Its message ("authentication failed") is part of the error chain, which
// autoshutdown.sh greps for.
var ErrAuthFailed = rcon.ErrAuthFailed

// IsAuthError reports whether err means the server rejected the password.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthFailed)
}

// Client wraps an RCON connection
type Client struct {
	conn *rcon.Conn
}

// NewClient creates a new RCON client using global config
func NewClient() (*Client, error) {
	cfg := config.Get()
	return NewClientWithConfig(cfg.Rcon.Host, cfg.Rcon.Port, cfg.Rcon.Password)
}

// NewClientWithConfig creates a new RCON client with explicit settings.
// A rejected password gives an error that satisfies IsAuthError, except in
// the rare case where closing the connection afterwards also fails: gorcon
// then returns ErrMultiErrorOccurred and the sentinel is lost.
func NewClientWithConfig(host string, port int, password string) (*Client, error) {
	if password == "" {
		return nil, fmt.Errorf("RCON password not configured")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := rcon.Dial(addr, password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RCON at %s: %w", addr, err)
	}

	return &Client{conn: conn}, nil
}

// Send executes a command via RCON and returns the response
func (c *Client) Send(command string) (string, error) {
	resp, err := c.conn.Execute(command)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	return resp, nil
}

// Close closes the RCON connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Status checks RCON connectivity and returns server status
func (c *Client) Status() (string, error) {
	// Try to list players as a status check
	return c.Send("list")
}
