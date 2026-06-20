package pulse

import (
	"sync"
	"testing"
	"time"

	"github.com/chun37/iidx-scratch-bridge/internal/rotation"
)

type event struct {
	vk   uint16
	down bool
}

type recorder struct {
	mu     sync.Mutex
	events []event
}

func (r *recorder) Send(vk uint16, down bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event{vk, down})
}

func (r *recorder) snapshot() []event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]event, len(r.events))
	copy(out, r.events)
	return out
}

const (
	upVK   = 0x26
	downVK = 0x28
)

func newRig(t *testing.T, dur time.Duration) (*Pulser, *recorder) {
	t.Helper()
	r := &recorder{}
	return New(dur, upVK, downVK, r.Send), r
}

func waitFor(t *testing.T, r *recorder, want int, deadline time.Duration) []event {
	t.Helper()
	start := time.Now()
	for time.Since(start) < deadline {
		if got := r.snapshot(); len(got) >= want {
			return got
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events; have %v", want, r.snapshot())
	return nil
}

func TestIdleToPressUp(t *testing.T) {
	p, r := newRig(t, 30*time.Millisecond)
	p.Rotate(rotation.DirUp)
	if got := r.snapshot(); len(got) != 1 || got[0] != (event{upVK, true}) {
		t.Fatalf("after Rotate(Up): %v", got)
	}
	if p.State() != PressingUp {
		t.Fatalf("state = %v, want PressingUp", p.State())
	}
}

func TestPulseReleasesAfterDuration(t *testing.T) {
	p, r := newRig(t, 20*time.Millisecond)
	p.Rotate(rotation.DirUp)
	events := waitFor(t, r, 2, 200*time.Millisecond)
	if events[1] != (event{upVK, false}) {
		t.Fatalf("expected up release, got %v", events[1])
	}
	if p.State() != Idle {
		t.Fatalf("state = %v, want Idle", p.State())
	}
}

func TestSameDirectionExtends(t *testing.T) {
	p, r := newRig(t, 30*time.Millisecond)
	p.Rotate(rotation.DirUp)
	time.Sleep(15 * time.Millisecond)
	p.Rotate(rotation.DirUp) // refresh
	time.Sleep(15 * time.Millisecond)
	// Still should be pressing (no release yet).
	if p.State() != PressingUp {
		t.Fatalf("state = %v, want PressingUp (timer extended)", p.State())
	}
	if got := r.snapshot(); len(got) != 1 {
		t.Fatalf("extension should not emit new events: %v", got)
	}
	// Now wait for full duration to elapse.
	time.Sleep(40 * time.Millisecond)
	events := waitFor(t, r, 2, 100*time.Millisecond)
	if events[1] != (event{upVK, false}) {
		t.Fatalf("expected up release, got %v", events[1])
	}
}

func TestReversalReleasesAndPresses(t *testing.T) {
	p, r := newRig(t, 100*time.Millisecond)
	p.Rotate(rotation.DirUp)
	p.Rotate(rotation.DirDown)
	got := r.snapshot()
	want := []event{
		{upVK, true},
		{upVK, false},
		{downVK, true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, ev := range want {
		if got[i] != ev {
			t.Fatalf("event %d: got %v, want %v", i, got[i], ev)
		}
	}
	if p.State() != PressingDown {
		t.Fatalf("state = %v, want PressingDown", p.State())
	}
}

func TestSupersededTimerDoesNotRelease(t *testing.T) {
	p, r := newRig(t, 20*time.Millisecond)
	p.Rotate(rotation.DirUp)
	// Reverse before the up timer can fire — the old timer must not
	// release the down key when it fires later.
	time.Sleep(5 * time.Millisecond)
	p.Rotate(rotation.DirDown)
	// Wait past the original up-timer deadline but well within the
	// fresh down-timer window.
	time.Sleep(15 * time.Millisecond)
	if p.State() != PressingDown {
		t.Fatalf("state = %v, want PressingDown (old timer should be ignored)", p.State())
	}
	// Eventually the down release fires.
	events := waitFor(t, r, 4, 200*time.Millisecond)
	if events[len(events)-1] != (event{downVK, false}) {
		t.Fatalf("last event = %v, want down release", events[len(events)-1])
	}
}

func TestContinuousRotationRepressesAfterDuration(t *testing.T) {
	// 30 ms duration. We feed Rotate(Up) every 12 ms for 144 ms, mimicking
	// 連皿: same-direction motion that should yield multiple keydown
	// events instead of one stuck key.
	p, r := newRig(t, 30*time.Millisecond)

	const tick = 12 * time.Millisecond
	const total = 144 * time.Millisecond
	deadline := time.Now().Add(total)
	for time.Now().Before(deadline) {
		p.Rotate(rotation.DirUp)
		time.Sleep(tick)
	}

	events := r.snapshot()
	presses := 0
	for _, e := range events {
		if e.vk == upVK && e.down {
			presses++
		}
	}
	// With 30 ms duration spread over 144 ms of continuous input, we
	// expect at least 3 distinct presses (the first one + at least two
	// re-presses). Anything less means the keydown is stuck.
	if presses < 3 {
		t.Fatalf("continuous rotation should produce >=3 presses, got %d (events=%v)", presses, events)
	}
}

func TestShortRotationDoesNotRepress(t *testing.T) {
	// One quick burst of same-direction taps shorter than duration must
	// stay as a single press to keep one-shot wheel motions clean.
	p, r := newRig(t, 100*time.Millisecond)
	p.Rotate(rotation.DirUp)
	time.Sleep(20 * time.Millisecond)
	p.Rotate(rotation.DirUp)
	time.Sleep(20 * time.Millisecond)
	p.Rotate(rotation.DirUp)

	events := r.snapshot()
	if len(events) != 1 || events[0] != (event{upVK, true}) {
		t.Fatalf("burst within duration should be a single press, got %v", events)
	}
}

func TestCloseReleasesHeldKey(t *testing.T) {
	p, r := newRig(t, time.Second)
	p.Rotate(rotation.DirUp)
	p.Close()
	got := r.snapshot()
	if len(got) != 2 || got[1] != (event{upVK, false}) {
		t.Fatalf("Close did not release: %v", got)
	}
	if p.State() != Idle {
		t.Fatalf("state = %v, want Idle", p.State())
	}
}
