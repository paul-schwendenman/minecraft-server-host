package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		envSetup map[string]string
		want     string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "absolute path unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde expands to home",
			path: "~",
			want: homeDir,
		},
		{
			name: "tilde with path",
			path: "~/Documents",
			want: filepath.Join(homeDir, "Documents"),
		},
		{
			name: "tilde with nested path",
			path: "~/path/to/file",
			want: filepath.Join(homeDir, "path/to/file"),
		},
		{
			name: "env var expansion",
			path: "$TEST_EXPAND_VAR/subdir",
			envSetup: map[string]string{
				"TEST_EXPAND_VAR": "/custom/path",
			},
			want: "/custom/path/subdir",
		},
		{
			name: "multiple env vars",
			path: "$TEST_VAR1/$TEST_VAR2",
			envSetup: map[string]string{
				"TEST_VAR1": "first",
				"TEST_VAR2": "second",
			},
			want: "first/second",
		},
		{
			name: "braced env var",
			path: "${TEST_BRACED_VAR}/path",
			envSetup: map[string]string{
				"TEST_BRACED_VAR": "/braced",
			},
			want: "/braced/path",
		},
		{
			name: "undefined env var becomes empty",
			path: "$UNDEFINED_TEST_VAR_12345/path",
			want: "/path",
		},
		{
			name: "tilde and env var combined",
			path: "~/$TEST_SUBDIR",
			envSetup: map[string]string{
				"TEST_SUBDIR": "mydir",
			},
			want: filepath.Join(homeDir, "mydir"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envSetup {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got := ExpandPath(tt.path)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExpandPathTildeNotAtStart(t *testing.T) {
	// Tilde in the middle of a path should NOT be expanded
	path := "/some/path/~user/file"
	got := ExpandPath(path)
	if got != path {
		t.Errorf("ExpandPath(%q) = %q, want %q (tilde should not expand mid-path)", path, got, path)
	}
}

func TestAbsPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		envSetup map[string]string
		want     string
		wantErr  bool
	}{
		{
			name: "absolute path stays absolute",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path becomes absolute",
			path: "relative",
			want: filepath.Join(cwd, "relative"),
		},
		{
			name: "tilde path becomes absolute",
			path: "~/Documents",
			want: filepath.Join(homeDir, "Documents"),
		},
		{
			name: "env var path becomes absolute",
			path: "$TEST_ABS_VAR/file",
			envSetup: map[string]string{
				"TEST_ABS_VAR": "/absolute",
			},
			want: "/absolute/file",
		},
		{
			name: "dot path resolves",
			path: "./file",
			want: filepath.Join(cwd, "file"),
		},
		{
			name: "parent path resolves",
			path: "../file",
			want: filepath.Join(filepath.Dir(cwd), "file"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envSetup {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got, err := AbsPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AbsPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("AbsPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAbsPathIsAbsolute(t *testing.T) {
	// Any path passed through AbsPath should result in an absolute path
	paths := []string{
		"relative",
		"./relative",
		"../parent",
		"~/home",
		"/already/absolute",
		".",
		"..",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			got, err := AbsPath(path)
			if err != nil {
				t.Fatalf("AbsPath(%q) returned error: %v", path, err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("AbsPath(%q) = %q, which is not absolute", path, got)
			}
		})
	}
}

func TestAbsPathNoDoubleSlashes(t *testing.T) {
	// Paths should be clean (no double slashes)
	paths := []string{
		"~/Documents",
		"/usr//local",
		"./test",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			got, err := AbsPath(path)
			if err != nil {
				t.Fatalf("AbsPath(%q) returned error: %v", path, err)
			}
			if strings.Contains(got, "//") {
				t.Errorf("AbsPath(%q) = %q, contains double slashes", path, got)
			}
		})
	}
}
