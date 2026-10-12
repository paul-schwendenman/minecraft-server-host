package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// SetKey rewrites every KEY=... line for key in the env file at path to
// value, leaving every other line (comments, blank lines, order) as it was.
// A value that was quoted keeps the same quote character, and an
// `export KEY=` prefix is kept. The file is replaced atomically (a temp file
// in the same directory, then rename) and keeps its mode and owner. It's an
// error if no line sets key: SetKey never appends one.
//
// value is written as is, so it must not contain a newline or the quote
// character the line uses.
func SetKey(path, key, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("value for %s contains a newline", key)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat env file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		newLine, ok, err := setLine(line, key, value)
		if err != nil {
			return err
		}
		if ok {
			lines[i] = newLine
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%s has no %s line", path, key)
	}
	return writeAtomic(path, []byte(strings.Join(lines, "\n")), info)
}

// setLine returns line with key's value replaced, and whether line sets key
func setLine(line, key, value string) (string, bool, error) {
	body := strings.TrimRight(line, "\r")
	cr := line[len(body):]
	trimmed := strings.TrimLeft(body, " \t")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return line, false, nil
	}
	idx := strings.Index(body, "=")
	if idx == -1 {
		return line, false, nil
	}
	name := strings.TrimSpace(body[:idx])
	name = strings.TrimSpace(strings.TrimPrefix(name, "export "))
	if name != key {
		return line, false, nil
	}

	rest := body[idx+1:]
	lead := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
	old := strings.TrimSpace(rest)
	quote := ""
	if len(old) >= 2 && (old[0] == '"' || old[0] == '\'') && old[len(old)-1] == old[0] {
		quote = old[:1]
	}
	if quote != "" && strings.Contains(value, quote) {
		return "", false, fmt.Errorf("value for %s contains its quote character %s", key, quote)
	}
	return body[:idx+1] + lead + quote + value + quote + cr, true, nil
}

// Backup copies the file at path to bakPath with mode 0600, replacing any
// earlier backup. The copy is owned by whoever runs this.
func Backup(path, bakPath string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	return writeAtomic(bakPath, data, nil)
}

// writeAtomic writes data to a temp file next to path and renames it over
// path. With like, the file gets like's mode and owner, otherwise mode 0600.
func writeAtomic(path string, data []byte, like os.FileInfo) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	mode := os.FileMode(0600)
	if like != nil {
		mode = like.Mode().Perm()
	}
	// CreateTemp makes the file 0600; chmod before writing so the content is
	// never readable with looser permissions than the target's
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set mode on %s: %w", tmpName, err)
	}
	if like != nil {
		if st, ok := like.Sys().(*syscall.Stat_t); ok {
			if err := tmp.Chown(int(st.Uid), int(st.Gid)); err != nil && !sameOwner(tmp, st) {
				tmp.Close()
				return fmt.Errorf("failed to keep the owner of %s: %w", path, err)
			}
		}
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to sync %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace %s: %w", path, err)
	}
	return nil
}

// sameOwner reports whether f already has st's owner, so a failed chown
// (not root) doesn't matter
func sameOwner(f *os.File, st *syscall.Stat_t) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	got, ok := info.Sys().(*syscall.Stat_t)
	return ok && got.Uid == st.Uid && got.Gid == st.Gid
}
