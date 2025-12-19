package rcon

import (
	"testing"
)

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

// Note: The following tests require a running Minecraft server with RCON enabled.
// They are skipped by default but can be run with:
//   RCON_HOST=localhost RCON_PORT=25575 RCON_PASSWORD=password go test -v -run Integration

func TestIntegrationSend(t *testing.T) {
	t.Skip("Integration test requires running Minecraft server with RCON")

	client, err := NewClientWithConfig("localhost", 25575, "test")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	resp, err := client.Send("list")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	t.Logf("Response: %s", resp)
}

func TestIntegrationStatus(t *testing.T) {
	t.Skip("Integration test requires running Minecraft server with RCON")

	client, err := NewClientWithConfig("localhost", 25575, "test")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	t.Logf("Status: %s", status)
}
