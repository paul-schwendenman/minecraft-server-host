package worlds

import (
	"errors"
	"reflect"
	"testing"
)

func TestEnableWorldTimers(t *testing.T) {
	origEnable, origEnableNow := enableUnit, enableUnitNow
	t.Cleanup(func() { enableUnit, enableUnitNow = origEnable, origEnableNow })

	var enabled, enabledNow []string
	enableUnit = func(unit string) error {
		enabled = append(enabled, unit)
		return nil
	}
	enableUnitNow = func(unit string) error {
		enabledNow = append(enabledNow, unit)
		// A failure on one timer mustn't stop the others
		return errors.New("not installed")
	}

	enableWorldTimers("survival")

	// map-build requires minecraft@survival.service, so starting it would
	// start the world: enable only
	if want := []string{"minecraft-map-build@survival.timer"}; !reflect.DeepEqual(enabled, want) {
		t.Errorf("enable = %v, want %v", enabled, want)
	}
	wantNow := []string{
		"minecraft-world-backup@survival.timer",
		"minecraft-map-backup@survival.timer",
	}
	if !reflect.DeepEqual(enabledNow, wantNow) {
		t.Errorf("enable --now = %v, want %v", enabledNow, wantNow)
	}
}
