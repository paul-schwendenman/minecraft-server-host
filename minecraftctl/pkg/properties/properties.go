package properties

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// entry represents a line in the properties file
type entry struct {
	key     string
	value   string
	comment string // comment preceding this key (including #)
	isBlank bool   // blank line
}

// Properties represents a Minecraft server.properties file
type Properties struct {
	entries []entry        // ordered list of entries (preserves file order)
	index   map[string]int // key -> index in entries
	path    string
}

// Load reads a server.properties file from disk
func Load(path string) (*Properties, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open properties file: %w", err)
	}
	defer file.Close()

	p := &Properties{
		entries: make([]entry, 0),
		index:   make(map[string]int),
		path:    path,
	}

	scanner := bufio.NewScanner(file)
	var pendingComment strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle blank lines
		if trimmed == "" {
			// If there are pending comments, save them as a comment-only entry first
			if pendingComment.Len() > 0 {
				p.entries = append(p.entries, entry{comment: pendingComment.String()})
				pendingComment.Reset()
			}
			p.entries = append(p.entries, entry{isBlank: true})
			continue
		}

		// Handle comments
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			if pendingComment.Len() > 0 {
				pendingComment.WriteString("\n")
			}
			pendingComment.WriteString(line)
			continue
		}

		// Parse key=value
		idx := strings.Index(line, "=")
		if idx == -1 {
			// Malformed line, treat as comment
			if pendingComment.Len() > 0 {
				pendingComment.WriteString("\n")
			}
			pendingComment.WriteString(line)
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := line[idx+1:] // Don't trim value - preserve leading/trailing spaces

		e := entry{
			key:     key,
			value:   value,
			comment: pendingComment.String(),
		}

		p.index[key] = len(p.entries)
		p.entries = append(p.entries, e)
		pendingComment.Reset()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading properties file: %w", err)
	}

	// Handle any trailing comments
	if pendingComment.Len() > 0 {
		p.entries = append(p.entries, entry{comment: pendingComment.String()})
	}

	return p, nil
}

// New creates an empty Properties
func New() *Properties {
	return &Properties{
		entries: make([]entry, 0),
		index:   make(map[string]int),
	}
}

// Get returns the value for a key
func (p *Properties) Get(key string) (string, bool) {
	if idx, ok := p.index[key]; ok {
		return p.entries[idx].value, true
	}
	return "", false
}

// GetInt returns an integer value for a key
func (p *Properties) GetInt(key string) (int, error) {
	val, ok := p.Get(key)
	if !ok {
		return 0, fmt.Errorf("key not found: %s", key)
	}
	return strconv.Atoi(val)
}

// GetBool returns a boolean value for a key
func (p *Properties) GetBool(key string) (bool, error) {
	val, ok := p.Get(key)
	if !ok {
		return false, fmt.Errorf("key not found: %s", key)
	}
	return strconv.ParseBool(val)
}

// Set sets a value for a key (adds if not exists)
func (p *Properties) Set(key, value string) {
	if idx, ok := p.index[key]; ok {
		p.entries[idx].value = value
	} else {
		p.index[key] = len(p.entries)
		p.entries = append(p.entries, entry{key: key, value: value})
	}
}

// SetInt sets an integer value
func (p *Properties) SetInt(key string, value int) {
	p.Set(key, strconv.Itoa(value))
}

// SetBool sets a boolean value
func (p *Properties) SetBool(key string, value bool) {
	p.Set(key, strconv.FormatBool(value))
}

// Delete removes a key
func (p *Properties) Delete(key string) {
	if idx, ok := p.index[key]; ok {
		// Mark entry as deleted by clearing the key
		p.entries[idx].key = ""
		delete(p.index, key)
	}
}

// Has checks if a key exists
func (p *Properties) Has(key string) bool {
	_, ok := p.index[key]
	return ok
}

// Keys returns all keys in order
func (p *Properties) Keys() []string {
	keys := make([]string, 0, len(p.index))
	for _, e := range p.entries {
		if e.key != "" {
			keys = append(keys, e.key)
		}
	}
	return keys
}

// Save writes the properties back to the original file
func (p *Properties) Save() error {
	if p.path == "" {
		return fmt.Errorf("no file path set")
	}
	return p.SaveTo(p.path)
}

// SaveTo writes the properties to a specific file
func (p *Properties) SaveTo(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create properties file: %w", err)
	}
	defer file.Close()

	for _, e := range p.entries {
		if e.isBlank {
			fmt.Fprintln(file)
			continue
		}

		if e.comment != "" {
			fmt.Fprintln(file, e.comment)
		}

		if e.key != "" {
			fmt.Fprintf(file, "%s=%s\n", e.key, e.value)
		}
	}

	return nil
}

// String returns the file content as a string
func (p *Properties) String() string {
	var sb strings.Builder

	for _, e := range p.entries {
		if e.isBlank {
			sb.WriteString("\n")
			continue
		}

		if e.comment != "" {
			sb.WriteString(e.comment)
			sb.WriteString("\n")
		}

		if e.key != "" {
			sb.WriteString(e.key)
			sb.WriteString("=")
			sb.WriteString(e.value)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Path returns the file path
func (p *Properties) Path() string {
	return p.path
}

// SetPath sets the file path
func (p *Properties) SetPath(path string) {
	p.path = path
}

// Len returns the number of key-value pairs
func (p *Properties) Len() int {
	return len(p.index)
}
