// Package status queries a running Minecraft server for the same details the
// web UI shows: version and the players currently online.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Tnze/go-mc/bot"
)

// Status is the subset of the server list ping response that we display.
type Status struct {
	Version       string
	PlayersOnline int
	PlayersMax    int
	// Players is the sample of online players reported by the server. Servers
	// may report fewer names than PlayersOnline.
	Players []string
}

// pingResponse mirrors the relevant parts of the server list ping JSON.
type pingResponse struct {
	Version struct {
		Name string `json:"name"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
		} `json:"sample"`
	} `json:"players"`
}

// Ping queries the server at addr (host:port) using the server list ping protocol.
func Ping(addr string, timeout time.Duration) (*Status, error) {
	raw, _, err := bot.PingAndListTimeout(addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to ping %s: %w", addr, err)
	}
	return parse(raw)
}

func parse(raw []byte) (*Status, error) {
	var resp pingResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	s := &Status{
		Version:       resp.Version.Name,
		PlayersOnline: resp.Players.Online,
		PlayersMax:    resp.Players.Max,
	}
	for _, p := range resp.Players.Sample {
		s.Players = append(s.Players, p.Name)
	}
	return s, nil
}

// Format renders the status using the same wording as the web UI.
func (s *Status) Format() string {
	var b strings.Builder

	if s.Version != "" {
		fmt.Fprintf(&b, "Minecraft version: %s\n", s.Version)
	}

	switch s.PlayersOnline {
	case 0:
		b.WriteString("The server has no active players.\n")
	case 1:
		b.WriteString("The server has 1 active player:\n")
	default:
		fmt.Fprintf(&b, "The server has %d active players:\n", s.PlayersOnline)
	}

	for _, name := range s.Players {
		fmt.Fprintf(&b, "  - %s\n", name)
	}

	return b.String()
}

const imdsBase = "http://169.254.169.254/latest"

// PublicIP returns the instance's public IPv4 address from EC2 instance
// metadata (IMDSv2). It returns an error when not running on EC2 or when the
// instance has no public address.
func PublicIP(timeout time.Duration) (string, error) {
	return publicIP(imdsBase, timeout)
}

func publicIP(base string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{}

	token, err := imdsRequest(ctx, client, http.MethodPut, base+"/api/token",
		map[string]string{"X-aws-ec2-metadata-token-ttl-seconds": "60"})
	if err != nil {
		return "", err
	}

	return imdsRequest(ctx, client, http.MethodGet, base+"/meta-data/public-ipv4",
		map[string]string{"X-aws-ec2-metadata-token": token})
}

func imdsRequest(ctx context.Context, client *http.Client, method, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("instance metadata returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// Addr joins host and port into a dialable address.
func Addr(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprint(port))
}
