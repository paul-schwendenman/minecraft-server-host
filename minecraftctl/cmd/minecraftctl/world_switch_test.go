package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/paul/minecraftctl/pkg/worlds"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{errors.New("boom"), 1},
		{fmt.Errorf("%w: bad name", worlds.ErrNotStartable), 2},
		{fmt.Errorf("%w: 2 players", worlds.ErrSwitchRefused), 3},
		{fmt.Errorf("%w: old", worlds.ErrStartFailed), 4},
		{fmt.Errorf("%w: old", worlds.ErrRollbackFailed), 5},
	}
	for _, tt := range tests {
		if got := exitCode(tt.err); got != tt.want {
			t.Errorf("exitCode(%v) = %d, want %d", tt.err, got, tt.want)
		}
	}
}
