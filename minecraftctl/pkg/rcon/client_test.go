package rcon

import (
	"net"
	"strconv"
	"strings"
	"testing"

	gorcon "github.com/gorcon/rcon"
	"github.com/gorcon/rcon/rcontest"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/spf13/viper"
)

// newTestServer starts a fake RCON server that accepts the password "right".
func newTestServer(t *testing.T) (string, int) {
	t.Helper()
	srv := rcontest.NewServer(rcontest.SetSettings(rcontest.Settings{Password: "right"}))
	t.Cleanup(srv.Close)
	host, portStr, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func TestNewClientWithConfigWrongPassword(t *testing.T) {
	host, port := newTestServer(t)
	_, err := NewClientWithConfig(host, port, "wrong")
	if err == nil {
		t.Fatal("Expected error for wrong password")
	}
	if !IsAuthError(err) {
		t.Errorf("IsAuthError(%v) = false, want true", err)
	}
	// autoshutdown.sh greps for this text.
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error %q doesn't mention \"authentication failed\"", err)
	}
}

func TestNewClientWithConfigRightPassword(t *testing.T) {
	host, port := newTestServer(t)
	client, err := NewClientWithConfig(host, port, "right")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestIsAuthErrorOtherErrors(t *testing.T) {
	if IsAuthError(nil) {
		t.Error("IsAuthError(nil) = true")
	}
	_, err := NewClientWithConfig("127.0.0.1", 59999, "password")
	if err == nil || IsAuthError(err) {
		t.Errorf("connection refused: IsAuthError(%v) = true", err)
	}
	_, err = NewClientWithConfig("localhost", 25575, "")
	if err == nil || IsAuthError(err) {
		t.Errorf("empty password: IsAuthError(%v) = true", err)
	}
}

func TestNewClientWithConfigEmptyPassword(t *testing.T) {
	// Should fail with empty password
	_, err := NewClientWithConfig("localhost", 25575, "")
	if err == nil {
		t.Error("Expected error for empty password")
	}
	if err.Error() != "RCON password not configured" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestNewClientWithConfigInvalidHost(t *testing.T) {
	// Should fail to connect to invalid host
	_, err := NewClientWithConfig("invalid.host.local", 25575, "password")
	if err == nil {
		t.Error("Expected error for invalid host")
	}
}

func TestNewClientWithConfigConnectionRefused(t *testing.T) {
	// Should fail when nothing is listening
	_, err := NewClientWithConfig("127.0.0.1", 59999, "password")
	if err == nil {
		t.Error("Expected error for connection refused")
	}
}

func TestClientStruct(t *testing.T) {
	// Test that Client struct can be created
	c := &Client{conn: nil}
	if c.conn != nil {
		t.Error("conn should be nil")
	}
}

func TestCloseNilConnection(t *testing.T) {
	// Close should handle nil connection gracefully
	c := &Client{conn: nil}
	err := c.Close()
	if err != nil {
		t.Errorf("Close with nil conn should not error: %v", err)
	}
}

// newEchoServer starts a fake RCON server (password "right") that answers
// "list" like Minecraft does and echoes anything else back.
func newEchoServer(t *testing.T) (string, int) {
	t.Helper()
	srv := rcontest.NewServer(
		rcontest.SetSettings(rcontest.Settings{Password: "right"}),
		rcontest.SetCommandHandler(func(c *rcontest.Context) {
			body := c.Request().Body()
			resp := "echo: " + body
			if body == "list" {
				resp = "There are 1 of a max of 20 players online: steve"
			}
			_, _ = gorcon.NewPacket(gorcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, resp).WriteTo(c.Conn())
		}),
	)
	t.Cleanup(srv.Close)
	host, portStr, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func TestSendAndStatus(t *testing.T) {
	host, port := newEchoServer(t)
	client, err := NewClientWithConfig(host, port, "right")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	resp, err := client.Send("say hello")
	if err != nil || resp != "echo: say hello" {
		t.Errorf("Send() = %q, %v; want echo", resp, err)
	}
	status, err := client.Status()
	if err != nil || !strings.HasPrefix(status, "There are 1 of a max") {
		t.Errorf("Status() = %q, %v; want the list reply", status, err)
	}
}

func TestSendBadReply(t *testing.T) {
	// The server answers with the wrong packet ID
	srv := rcontest.NewServer(
		rcontest.SetSettings(rcontest.Settings{Password: "right"}),
		rcontest.SetCommandHandler(func(c *rcontest.Context) {
			_, _ = gorcon.NewPacket(gorcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID+1, "?").WriteTo(c.Conn())
		}),
	)
	t.Cleanup(srv.Close)
	host, portStr, _ := net.SplitHostPort(srv.Addr())
	port, _ := strconv.Atoi(portStr)
	client, err := NewClientWithConfig(host, port, "right")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Send("list"); err == nil || !strings.Contains(err.Error(), "failed to execute command") {
		t.Errorf("Send() with a bad reply = %v, want an execute error", err)
	}
}

func TestNewClientUsesConfig(t *testing.T) {
	host, port := newEchoServer(t)
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("RCON_HOST", host)
	t.Setenv("RCON_PORT", strconv.Itoa(port))
	t.Setenv("RCON_PASSWORD", "right")
	if err := config.Init(""); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() = %v", err)
	}
	defer client.Close()
	if resp, err := client.Send("ping"); err != nil || resp != "echo: ping" {
		t.Errorf("Send() = %q, %v", resp, err)
	}
}
