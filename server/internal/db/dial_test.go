package db

import (
	"runtime"
	"testing"
)

// TestDialWorkersMapsThePresets pins the dial's three steps: light
// runs one, standard runs half the cores, full runs all but one --
// floored at one, and capped so the mapping never promises more
// workers than the pool's ceiling carries.
func TestDialWorkersMapsThePresets(t *testing.T) {
	if got := DialWorkers("light"); got != 1 {
		t.Errorf("DialWorkers(light) = %d, want 1", got)
	}
	cores := runtime.NumCPU()
	if want := max(1, min(cores/2, dialCeiling)); DialWorkers("standard") != want {
		t.Errorf("DialWorkers(standard) = %d, want %d", DialWorkers("standard"), want)
	}
	if want := max(1, min(cores-1, dialCeiling)); DialWorkers("full") != want {
		t.Errorf("DialWorkers(full) = %d, want %d", DialWorkers("full"), want)
	}
	if got := DialWorkers("turbo"); got != 1 {
		t.Errorf("DialWorkers(unknown) = %d, want the light step's 1", got)
	}
}
