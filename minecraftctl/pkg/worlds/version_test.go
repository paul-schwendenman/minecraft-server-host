package worlds

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		// Equal versions
		{name: "equal simple", v1: "1.20", v2: "1.20", want: 0},
		{name: "equal three parts", v1: "1.20.1", v2: "1.20.1", want: 0},
		{name: "equal with trailing zero", v1: "1.20.0", v2: "1.20.0", want: 0},

		// v1 < v2
		{name: "major less", v1: "1.20.1", v2: "2.0.0", want: -1},
		{name: "minor less", v1: "1.19.4", v2: "1.20.1", want: -1},
		{name: "patch less", v1: "1.20.1", v2: "1.20.2", want: -1},
		{name: "multi-digit patch less", v1: "1.21.1", v2: "1.21.11", want: -1},

		// v1 > v2
		{name: "major greater", v1: "2.0.0", v2: "1.20.1", want: 1},
		{name: "minor greater", v1: "1.21.0", v2: "1.20.4", want: 1},
		{name: "patch greater", v1: "1.20.4", v2: "1.20.1", want: 1},
		{name: "multi-digit patch greater", v1: "1.21.11", v2: "1.21.1", want: 1},

		// Different length versions
		{name: "shorter equals longer with zero", v1: "1.20", v2: "1.20.0", want: 0},
		{name: "shorter less than longer", v1: "1.20", v2: "1.20.1", want: -1},
		{name: "shorter greater major", v1: "1.21", v2: "1.20.4", want: 1},
		{name: "longer less than shorter", v1: "1.20.0", v2: "1.21", want: -1},

		// Real Minecraft version examples
		{name: "minecraft 1.19.4 vs 1.20", v1: "1.19.4", v2: "1.20", want: -1},
		{name: "minecraft 1.20 vs 1.20.1", v1: "1.20", v2: "1.20.1", want: -1},
		{name: "minecraft 1.20.4 vs 1.21", v1: "1.20.4", v2: "1.21", want: -1},

		// Error cases
		{name: "invalid v1", v1: "invalid", v2: "1.20.1", wantErr: true},
		{name: "invalid v2", v1: "1.20.1", v2: "invalid", wantErr: true},
		{name: "empty v1", v1: "", v2: "1.20.1", wantErr: true},
		{name: "empty v2", v1: "1.20.1", v2: "", wantErr: true},
		{name: "non-numeric component", v1: "1.20.a", v2: "1.20.1", wantErr: true},
		{name: "mixed valid invalid", v1: "1.x.1", v2: "1.20.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions(%q, %q) error = %v, wantErr %v", tt.v1, tt.v2, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestCompareVersionsSymmetry(t *testing.T) {
	// Test that CompareVersions(a, b) == -CompareVersions(b, a)
	versions := []string{"1.20", "1.20.1", "1.21", "1.19.4", "1.21.11"}

	for _, v1 := range versions {
		for _, v2 := range versions {
			t.Run(v1+"_vs_"+v2, func(t *testing.T) {
				result1, err1 := CompareVersions(v1, v2)
				result2, err2 := CompareVersions(v2, v1)

				if err1 != nil || err2 != nil {
					t.Fatalf("Unexpected error: err1=%v, err2=%v", err1, err2)
				}

				if result1 != -result2 {
					t.Errorf("Symmetry violated: CompareVersions(%q, %q)=%d, CompareVersions(%q, %q)=%d",
						v1, v2, result1, v2, v1, result2)
				}
			})
		}
	}
}

func TestCompareVersionsTransitivity(t *testing.T) {
	// Test that if a < b and b < c, then a < c
	orderedVersions := []string{"1.19", "1.19.4", "1.20", "1.20.1", "1.21", "1.21.1", "1.21.11"}

	for i := 0; i < len(orderedVersions); i++ {
		for j := i + 1; j < len(orderedVersions); j++ {
			t.Run(orderedVersions[i]+"_lt_"+orderedVersions[j], func(t *testing.T) {
				result, err := CompareVersions(orderedVersions[i], orderedVersions[j])
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result != -1 {
					t.Errorf("Expected %q < %q, got result %d",
						orderedVersions[i], orderedVersions[j], result)
				}
			})
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    []int
		wantErr bool
	}{
		{name: "two parts", version: "1.20", want: []int{1, 20}},
		{name: "three parts", version: "1.20.1", want: []int{1, 20, 1}},
		{name: "four parts", version: "1.2.3.4", want: []int{1, 2, 3, 4}},
		{name: "single part", version: "1", want: []int{1}},
		{name: "multi-digit", version: "1.21.11", want: []int{1, 21, 11}},
		{name: "zeros", version: "0.0.0", want: []int{0, 0, 0}},

		// Error cases
		{name: "empty", version: "", wantErr: true},
		{name: "letters", version: "a.b.c", wantErr: true},
		{name: "mixed", version: "1.a.2", wantErr: true},
		{name: "float", version: "1.20.1.5", want: []int{1, 20, 1, 5}}, // This is valid, just 4 parts
		{name: "spaces", version: "1. 20.1", wantErr: true},

		// Note: negative numbers are accepted by strconv.Atoi but aren't valid Minecraft versions
		{name: "negative", version: "-1.20.1", want: []int{-1, 20, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parseVersion(%q) = %v, want %v", tt.version, got, tt.want)
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("parseVersion(%q) = %v, want %v", tt.version, got, tt.want)
						return
					}
				}
			}
		})
	}
}
