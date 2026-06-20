package rotation

import "testing"

func TestWrappedDelta(t *testing.T) {
	tests := []struct {
		prev, curr uint8
		want       int
	}{
		{254, 2, 4},
		{2, 254, -4},
		{100, 105, 5},
		{100, 95, -5},
		{0, 0, 0},
		// d == ±128 is ambiguous (shortest path is the same either
		// way). The implementation only wraps when |d| > 128.
		{0, 128, 128},
		{128, 0, -128},
		{0, 127, 127},
		{127, 0, -127},
	}
	for _, tt := range tests {
		got := WrappedDelta(tt.prev, tt.curr)
		if got != tt.want {
			t.Errorf("WrappedDelta(%d,%d) = %d, want %d", tt.prev, tt.curr, got, tt.want)
		}
	}
}

func TestDetectorFirstSampleSilent(t *testing.T) {
	d := New(2)
	if got := d.Update(100); got != DirNone {
		t.Errorf("first sample fired %d", got)
	}
}

func TestDetectorFiresAboveThreshold(t *testing.T) {
	d := New(2)
	d.Update(100)
	if got := d.Update(103); got != DirUp {
		t.Errorf("delta +3 fired %d, want DirUp", got)
	}
	if got := d.Update(100); got != DirDown {
		t.Errorf("delta -3 fired %d, want DirDown", got)
	}
}

func TestDetectorAccumulatesSubThreshold(t *testing.T) {
	d := New(3)
	d.Update(0)
	// Each +1 step is below threshold but they should accumulate.
	if got := d.Update(1); got != DirNone {
		t.Errorf("step 1 fired %d", got)
	}
	if got := d.Update(2); got != DirNone {
		t.Errorf("step 2 fired %d", got)
	}
	if got := d.Update(3); got != DirUp {
		t.Errorf("step 3 (accum=3) fired %d, want DirUp", got)
	}
	// Accumulator reset to 0 after firing.
	if got := d.Update(4); got != DirNone {
		t.Errorf("post-fire step fired %d", got)
	}
}

func TestDetectorWrapAround(t *testing.T) {
	d := New(2)
	d.Update(254)
	if got := d.Update(2); got != DirUp {
		t.Errorf("254->2 fired %d, want DirUp", got)
	}
}

func TestDetectorReset(t *testing.T) {
	d := New(2)
	d.Update(100)
	d.Update(101) // accumulates +1
	d.Reset()
	if got := d.Update(200); got != DirNone {
		t.Errorf("first sample after reset fired %d", got)
	}
	if got := d.Update(203); got != DirUp {
		t.Errorf("delta after reset fired %d, want DirUp", got)
	}
}
