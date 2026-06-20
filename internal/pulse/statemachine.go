package pulse

import (
	"sync"
	"time"

	"github.com/chun37/iidx-scratch-bridge/internal/rotation"
)

// State is the pulse-emission state for one scratch axis.
type State int

const (
	Idle State = iota
	PressingUp
	PressingDown
)

// KeyAction is the callback that delivers a key press/release to the
// outside world. Implementations should be fast and side-effect-only.
type KeyAction func(vk uint16, down bool)

// Pulser turns Direction events from the Detector into long-press key
// pulses that mimic a stock IIDX controller's 74HC monostable.
type Pulser struct {
	duration time.Duration
	upVK     uint16
	downVK   uint16
	send     KeyAction

	mu    sync.Mutex
	state State
	// generation is bumped on every state change so a fired timer
	// can tell whether it is still authoritative.
	generation uint64
	timer      *time.Timer
}

// New returns a Pulser ready to receive Rotate events. duration is the
// length of each key press; the design doc calls for 133 ms.
func New(duration time.Duration, upVK, downVK uint16, send KeyAction) *Pulser {
	return &Pulser{
		duration: duration,
		upVK:     upVK,
		downVK:   downVK,
		send:     send,
	}
}

// State returns the current state (for tests / observability).
func (p *Pulser) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Rotate feeds a rotation event into the state machine.
func (p *Pulser) Rotate(dir rotation.Direction) {
	if dir == rotation.DirNone {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case Idle:
		if dir == rotation.DirUp {
			p.send(p.upVK, true)
			p.state = PressingUp
		} else {
			p.send(p.downVK, true)
			p.state = PressingDown
		}
		p.armTimer()

	case PressingUp:
		if dir == rotation.DirUp {
			// Same direction: just extend the pulse.
			p.armTimer()
		} else {
			// Reversal: release old, press new.
			p.send(p.upVK, false)
			p.send(p.downVK, true)
			p.state = PressingDown
			p.armTimer()
		}

	case PressingDown:
		if dir == rotation.DirDown {
			p.armTimer()
		} else {
			p.send(p.downVK, false)
			p.send(p.upVK, true)
			p.state = PressingUp
			p.armTimer()
		}
	}
}

// Close releases any currently-held key. Safe to call multiple times.
func (p *Pulser) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	switch p.state {
	case PressingUp:
		p.send(p.upVK, false)
	case PressingDown:
		p.send(p.downVK, false)
	}
	p.state = Idle
	p.generation++
}

// armTimer must be called with p.mu held. It (re)starts the timer for
// the current state.
func (p *Pulser) armTimer() {
	if p.timer != nil {
		p.timer.Stop()
	}
	p.generation++
	gen := p.generation
	p.timer = time.AfterFunc(p.duration, func() {
		p.expire(gen)
	})
}

// expire is the timer callback. It releases the held key only if no
// further rotation has happened since the timer was armed.
func (p *Pulser) expire(gen uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.generation != gen {
		// Superseded by a later Rotate; do nothing.
		return
	}
	switch p.state {
	case PressingUp:
		p.send(p.upVK, false)
	case PressingDown:
		p.send(p.downVK, false)
	}
	p.state = Idle
	p.timer = nil
}
