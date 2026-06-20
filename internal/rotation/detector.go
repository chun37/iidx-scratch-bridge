package rotation

// Direction encodes the sign of a scratch rotation event.
type Direction int

const (
	DirNone Direction = 0
	DirUp   Direction = +1
	DirDown Direction = -1
)

// WrappedDelta computes the shortest signed delta between two 8-bit
// position samples, treating the value space as a ring.
//
// 254 -> 2  yields +4 (forward through 0), not -252.
// 2   -> 254 yields -4 (backward through 0), not +252.
func WrappedDelta(prev, curr uint8) int {
	d := int(curr) - int(prev)
	switch {
	case d > 128:
		return d - 256
	case d < -128:
		return d + 256
	default:
		return d
	}
}

// Detector accumulates sub-threshold motion so slow rotations still
// emit events once enough delta has piled up in one direction.
type Detector struct {
	threshold   int
	prev        uint8
	accumulated int
	initialized bool
}

// New returns a Detector with the given firing threshold.
// threshold must be >= 1.
func New(threshold int) *Detector {
	if threshold < 1 {
		threshold = 1
	}
	return &Detector{threshold: threshold}
}

// Reset clears all accumulated state. Call after a long stall or after
// the device reconnects so a stale prev value does not cause a phantom
// jump on the first new sample.
func (d *Detector) Reset() {
	d.prev = 0
	d.accumulated = 0
	d.initialized = false
}

// Update feeds the next axis sample and returns DirUp / DirDown when
// the accumulated motion crosses the threshold, or DirNone otherwise.
// The first sample after construction or Reset only seeds the state.
func (d *Detector) Update(curr uint8) Direction {
	if !d.initialized {
		d.prev = curr
		d.initialized = true
		return DirNone
	}
	delta := WrappedDelta(d.prev, curr)
	d.prev = curr
	d.accumulated += delta

	if d.accumulated >= d.threshold {
		d.accumulated = 0
		return DirUp
	}
	if d.accumulated <= -d.threshold {
		d.accumulated = 0
		return DirDown
	}
	return DirNone
}
