package systemd

import (
	"testing"
)

func TestFormatUnitName(t *testing.T) {
	tests := []struct {
		prefix   string
		instance string
		unitType UnitType
		want     string
	}{
		{"minecraft", "survival", UnitService, "minecraft@survival.service"},
		{"minecraft", "creative", UnitService, "minecraft@creative.service"},
		{"minecraft-world-backup", "survival", UnitService, "minecraft-world-backup@survival.service"},
		{"minecraft-world-backup", "test", UnitTimer, "minecraft-world-backup@test.timer"},
		{"minecraft-map-backup", "world", UnitService, "minecraft-map-backup@world.service"},
		{"minecraft-map-update", "survival", UnitTimer, "minecraft-map-update@survival.timer"},
		{"minecraft-map-update", "creative", UnitService, "minecraft-map-update@creative.service"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatUnitName(tt.prefix, tt.instance, tt.unitType)
			if got != tt.want {
				t.Errorf("FormatUnitName(%q, %q, %q) = %q, want %q",
					tt.prefix, tt.instance, tt.unitType, got, tt.want)
			}
		})
	}
}

func TestUnitTypeConstants(t *testing.T) {
	if UnitService != "service" {
		t.Errorf("UnitService = %q, want %q", UnitService, "service")
	}
	if UnitTimer != "timer" {
		t.Errorf("UnitTimer = %q, want %q", UnitTimer, "timer")
	}
}
