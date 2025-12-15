package worlds

import (
	"fmt"
	"strconv"
	"strings"
)

// CompareVersions compares two Minecraft version strings.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
// Handles versions like "1.21.1", "1.21.11", "1.20".
func CompareVersions(v1, v2 string) (int, error) {
	parts1, err := parseVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v1, err)
	}
	parts2, err := parseVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v2, err)
	}

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1 = parts1[i]
		}
		if i < len(parts2) {
			p2 = parts2[i]
		}

		if p1 < p2 {
			return -1, nil
		}
		if p1 > p2 {
			return 1, nil
		}
	}

	return 0, nil
}

func parseVersion(version string) ([]int, error) {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("non-numeric version component: %s", part)
		}
		result[i] = n
	}
	return result, nil
}
