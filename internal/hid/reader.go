// Package hid reads Input Reports from a USB HID device and decodes
// them into ScratchAxis + Buttons fields per the user's config.
package hid

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bearsh/hid"
)

// InputReport is the decoded subset of an HID Input Report the bridge
// cares about. ScratchAxis is the raw 8-bit wheel position; Buttons is
// the packed button bitmap (little-endian over the configured range).
type InputReport struct {
	ScratchAxis uint8
	Buttons     uint16
}

// Parser turns a raw byte slice into an InputReport based on the
// configured layout. Exposed so the main loop can reuse it without
// going through HID I/O for tests.
type Parser struct {
	ScratchIdx int
	BtnStart   int
	BtnEnd     int // exclusive
	ReportID   int // 0 = no report ID
}

// Parse extracts an InputReport from a raw byte slice. Returns ok=false
// for too-short or ReportID-mismatched buffers; the caller should drop
// such reports silently.
func (p Parser) Parse(buf []byte) (InputReport, bool) {
	if p.ReportID != 0 {
		if len(buf) == 0 || int(buf[0]) != p.ReportID {
			return InputReport{}, false
		}
	}
	if p.ScratchIdx < 0 || p.ScratchIdx >= len(buf) {
		return InputReport{}, false
	}
	if p.BtnStart < 0 || p.BtnEnd > len(buf) || p.BtnEnd <= p.BtnStart {
		return InputReport{}, false
	}
	var buttons uint16
	switch p.BtnEnd - p.BtnStart {
	case 1:
		buttons = uint16(buf[p.BtnStart])
	case 2:
		buttons = uint16(buf[p.BtnStart]) | uint16(buf[p.BtnStart+1])<<8
	default:
		return InputReport{}, false
	}
	return InputReport{
		ScratchAxis: buf[p.ScratchIdx],
		Buttons:     buttons,
	}, true
}

// Reader opens an HID device, reads Input Reports in a loop, decodes
// them, and pushes the result to a callback.
type Reader struct {
	VID      uint16
	PID      uint16
	Parser   Parser
	OnReport func(InputReport)
	// OnRawReport, if set, is invoked for every received Input Report
	// *before* the Parser runs and regardless of whether parsing would
	// succeed. The slice is only valid until OnRawReport returns; copy
	// if you need to retain it. Used by --dump mode so the user can
	// discover the right offsets without already knowing them.
	OnRawReport func([]byte)
	// OnReconnect, if set, is called whenever the device is (re)opened.
	// Use this to reset any stateful detectors.
	OnReconnect func()
	// Logger, if set, receives lifecycle messages.
	Logger *log.Logger
}

func (r *Reader) logf(format string, args ...interface{}) {
	if r.Logger != nil {
		r.Logger.Printf(format, args...)
	}
}

// Run blocks until ctx is cancelled. On any error it logs, sleeps for
// reconnectDelay, and tries again.
func (r *Reader) Run(ctx context.Context) {
	const reconnectDelay = time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			r.logf("hid: %v (retrying in %s)", err, reconnectDelay)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (r *Reader) runOnce(ctx context.Context) error {
	infos := hid.Enumerate(r.VID, r.PID)
	if len(infos) == 0 {
		return fmt.Errorf("no HID device matches VID=0x%04X PID=0x%04X", r.VID, r.PID)
	}
	// If the device exposes multiple interfaces (common for controllers
	// that combine a joystick interface with a config interface),
	// prefer the lowest-numbered interface. Empirically the gameplay
	// interface is the first one on Konami-style entry models.
	chosen := infos[0]
	for _, info := range infos[1:] {
		if info.Interface < chosen.Interface {
			chosen = info
		}
	}

	dev, err := chosen.Open()
	if err != nil {
		return fmt.Errorf("open device: %w", err)
	}
	defer dev.Close()

	r.logf("hid: opened %q (iface=%d, path=%s)", chosen.Product, chosen.Interface, chosen.Path)
	if r.OnReconnect != nil {
		r.OnReconnect()
	}

	buf := make([]byte, 64)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := dev.ReadTimeout(buf, 100)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		if n == 0 {
			continue
		}
		if r.OnRawReport != nil {
			r.OnRawReport(buf[:n])
		}
		if r.OnReport != nil {
			report, ok := r.Parser.Parse(buf[:n])
			if !ok {
				continue
			}
			r.OnReport(report)
		}
	}
}
