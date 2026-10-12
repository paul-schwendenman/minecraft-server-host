package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gorcon "github.com/gorcon/rcon"
	"github.com/gorcon/rcon/rcontest"
)

// startFakeRcon starts an in-process RCON server, points the config at it
// and returns the commands it received.
func startFakeRcon(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var got []string
	srv := rcontest.NewServer(
		rcontest.SetSettings(rcontest.Settings{Password: "secret"}),
		rcontest.SetCommandHandler(func(c *rcontest.Context) {
			cmd := c.Request().Body()
			mu.Lock()
			got = append(got, cmd)
			mu.Unlock()
			resp := ""
			switch {
			case cmd == "list":
				resp = "There are 0 of a max of 20 players online:"
			case strings.HasPrefix(cmd, "say "):
				resp = "said: " + strings.TrimPrefix(cmd, "say ")
			}
			_, _ = gorcon.NewPacket(gorcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, resp).WriteTo(c.Conn())
		}),
	)
	t.Cleanup(srv.Close)
	host, port, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RCON_HOST", host)
	t.Setenv("RCON_PORT", port)
	t.Setenv("RCON_PASSWORD", "secret")
	setupWorldTestConfig(t, t.TempDir())
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

func TestRconStatusAndSendCmds(t *testing.T) {
	received := startFakeRcon(t)

	out, err := captureWorldStdout(t, func() error { return rconStatusCmd.RunE(rconStatusCmd, nil) })
	if err != nil || out != "There are 0 of a max of 20 players online:\n" {
		t.Errorf("rcon status = %q, %v", out, err)
	}
	out, err = captureWorldStdout(t, func() error {
		return rconSendCmd.RunE(rconSendCmd, []string{"say", "hello", "world"})
	})
	if err != nil || out != "said: hello world\n" {
		t.Errorf("rcon send = %q, %v", out, err)
	}
	if got := strings.Join(received(), "|"); got != "list|say hello world" {
		t.Errorf("server received %q", got)
	}
}

func TestRconExecCmd(t *testing.T) {
	received := startFakeRcon(t)
	file := filepath.Join(t.TempDir(), "cmds.txt")
	script := "# set up\nsay one\n\n  time set day  \n# done\nsay two\n"
	if err := os.WriteFile(file, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := captureWorldStdout(t, func() error { return rconExecCmd.RunE(rconExecCmd, []string{file}) })
	if err != nil {
		t.Fatalf("rcon exec = %v", err)
	}
	// Empty replies aren't printed
	if out != "said: one\nsaid: two\n" {
		t.Errorf("output = %q", out)
	}
	if got := strings.Join(received(), "|"); got != "say one|time set day|say two" {
		t.Errorf("server received %q (comments and blanks should be skipped)", got)
	}

	if err := rconExecCmd.RunE(rconExecCmd, []string{filepath.Join(t.TempDir(), "missing")}); err == nil || !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("rcon exec of a missing file = %v", err)
	}
}

func TestRconCmdsNoPassword(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "")
	t.Setenv("MINECRAFT_RCON_PASSWORD", "")
	setupWorldTestConfig(t, t.TempDir())
	file := filepath.Join(t.TempDir(), "cmds.txt")
	if err := os.WriteFile(file, []byte("list\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for name, run := range map[string]func() error{
		"status": func() error { return rconStatusCmd.RunE(rconStatusCmd, nil) },
		"send":   func() error { return rconSendCmd.RunE(rconSendCmd, []string{"list"}) },
		"exec":   func() error { return rconExecCmd.RunE(rconExecCmd, []string{file}) },
	} {
		if err := run(); err == nil || !strings.Contains(err.Error(), "password not configured") {
			t.Errorf("rcon %s without a password = %v", name, err)
		}
	}
}
