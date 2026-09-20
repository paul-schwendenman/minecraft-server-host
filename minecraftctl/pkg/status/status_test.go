package status

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	raw := []byte(`{
		"version": {"name": "1.21.4", "protocol": 769},
		"players": {"max": 20, "online": 2, "sample": [
			{"name": "alice", "id": "1"},
			{"name": "bob", "id": "2"}
		]},
		"description": {"text": "hi"}
	}`)

	s, err := parse(raw)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if string(s.Raw) != string(raw) {
		t.Errorf("Raw = %s, want the unmodified response", s.Raw)
	}
	s.Raw = nil

	want := &Status{Version: "1.21.4", PlayersOnline: 2, PlayersMax: 20, Players: []string{"alice", "bob"}}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("parse() = %+v, want %+v", s, want)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := parse([]byte("not json")); err == nil {
		t.Error("parse() expected error for invalid JSON")
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		s    Status
		want string
	}{
		{
			name: "no players",
			s:    Status{Version: "1.21.4"},
			want: "Minecraft version: 1.21.4\nThe server has no active players.\n",
		},
		{
			name: "one player",
			s:    Status{Version: "1.21.4", PlayersOnline: 1, Players: []string{"alice"}},
			want: "Minecraft version: 1.21.4\nThe server has 1 active player:\n  - alice\n",
		},
		{
			name: "many players",
			s:    Status{Version: "1.21.4", PlayersOnline: 2, Players: []string{"alice", "bob"}},
			want: "Minecraft version: 1.21.4\nThe server has 2 active players:\n  - alice\n  - bob\n",
		},
		{
			name: "no version",
			s:    Status{},
			want: "The server has no active players.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Format(); got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublicIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/token":
			w.Write([]byte("tok"))
		case r.URL.Path == "/meta-data/public-ipv4" && r.Header.Get("X-aws-ec2-metadata-token") == "tok":
			w.Write([]byte("203.0.113.7\n"))
		default:
			http.Error(w, "nope", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ip, err := publicIP(srv.URL, time.Second)
	if err != nil {
		t.Fatalf("publicIP() error = %v", err)
	}
	if ip != "203.0.113.7" {
		t.Errorf("publicIP() = %q, want 203.0.113.7", ip)
	}
}

func TestPublicIPNoPublicAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.Write([]byte("tok"))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := publicIP(srv.URL, time.Second); err == nil {
		t.Error("publicIP() expected error when instance has no public IPv4")
	}
}

func TestAddr(t *testing.T) {
	if got := Addr("127.0.0.1", 25565); got != "127.0.0.1:25565" {
		t.Errorf("Addr() = %q", got)
	}
	if got := Addr("::1", 25565); got != "[::1]:25565" {
		t.Errorf("Addr() = %q", got)
	}
}

func TestNewReportRunning(t *testing.T) {
	raw := `{"version":{"name":"1.21.4"},"players":{"max":20,"online":1,"sample":[{"name":"alice","id":"a"}]}}`
	s, err := parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}

	out, err := json.Marshal(NewReport(s, nil, "203.0.113.7"))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"instance":{"state":"running","ip_address":"203.0.113.7"},` +
		`"dns_record":{"name":null,"value":null,"type":null},"details":` + raw + `}`
	if string(out) != want {
		t.Errorf("report = %s\nwant     %s", out, want)
	}
}

func TestNewReportNotRunning(t *testing.T) {
	out, err := json.Marshal(NewReport(nil, errors.New("connection refused"), ""))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"instance":{"state":"stopped","ip_address":null},` +
		`"dns_record":{"name":null,"value":null,"type":null},"details":null,"error":"connection refused"}`
	if string(out) != want {
		t.Errorf("report = %s\nwant     %s", out, want)
	}
}
