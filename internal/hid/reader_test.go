package hid

import (
	"testing"

	"github.com/bearsh/hid"
)

func TestPickGameCollectionPrefersJoystick(t *testing.T) {
	vendor := hid.DeviceInfo{UsagePage: 0xFF00, Usage: 0x01, Interface: 0, Product: "Entry Model"}
	joystick := hid.DeviceInfo{UsagePage: 0x01, Usage: 0x04, Interface: 0, Product: ""}
	// Vendor-specific collection appears first in the list (as observed
	// on Windows hidapi in some hot-plug orders). The picker must still
	// pull the Joystick collection.
	got := pickGameCollection([]hid.DeviceInfo{vendor, joystick})
	if got.UsagePage != 0x01 || got.Usage != 0x04 {
		t.Fatalf("got %+v, want Joystick collection", got)
	}
}

func TestPickGameCollectionPrefersGamepad(t *testing.T) {
	gamepad := hid.DeviceInfo{UsagePage: 0x01, Usage: 0x05, Interface: 0}
	vendor := hid.DeviceInfo{UsagePage: 0xFF00, Usage: 0x01, Interface: 0}
	got := pickGameCollection([]hid.DeviceInfo{vendor, gamepad})
	if got.UsagePage != 0x01 || got.Usage != 0x05 {
		t.Fatalf("got %+v, want Gamepad collection", got)
	}
}

func TestPickGameCollectionFallsBackToLowestInterface(t *testing.T) {
	// No Generic Desktop joystick/gamepad at all -- pick the lowest
	// interface number as a sane default.
	a := hid.DeviceInfo{UsagePage: 0xFF00, Usage: 0x01, Interface: 1}
	b := hid.DeviceInfo{UsagePage: 0xFF00, Usage: 0x02, Interface: 0}
	got := pickGameCollection([]hid.DeviceInfo{a, b})
	if got.Interface != 0 {
		t.Fatalf("got iface=%d, want 0", got.Interface)
	}
}

func TestPickGameCollectionSingleEntry(t *testing.T) {
	only := hid.DeviceInfo{UsagePage: 0x01, Usage: 0x04, Interface: 0}
	got := pickGameCollection([]hid.DeviceInfo{only})
	if got != only {
		t.Fatalf("single-entry picker returned a different value: %+v", got)
	}
}

func TestParserBasic(t *testing.T) {
	p := Parser{ScratchIdx: 5, BtnStart: 2, BtnEnd: 4}
	buf := []byte{0x00, 0x00, 0x12, 0x34, 0x00, 0xAB, 0x00}
	got, ok := p.Parse(buf)
	if !ok {
		t.Fatal("parse failed")
	}
	if got.ScratchAxis != 0xAB {
		t.Errorf("ScratchAxis = %#x", got.ScratchAxis)
	}
	// little-endian: low byte first
	if got.Buttons != 0x3412 {
		t.Errorf("Buttons = %#x, want 0x3412", got.Buttons)
	}
}

func TestParserSingleByteButtons(t *testing.T) {
	p := Parser{ScratchIdx: 1, BtnStart: 0, BtnEnd: 1}
	buf := []byte{0xF0, 0x55}
	got, ok := p.Parse(buf)
	if !ok {
		t.Fatal("parse failed")
	}
	if got.Buttons != 0x00F0 || got.ScratchAxis != 0x55 {
		t.Errorf("got %+v", got)
	}
}

func TestParserReportIDMismatch(t *testing.T) {
	p := Parser{ScratchIdx: 5, BtnStart: 2, BtnEnd: 4, ReportID: 0x01}
	// First byte 0x02 != 0x01 -> drop.
	buf := []byte{0x02, 0x00, 0x12, 0x34, 0x00, 0xAB}
	if _, ok := p.Parse(buf); ok {
		t.Fatal("expected drop on report ID mismatch")
	}
	// Correct ID accepted.
	buf2 := []byte{0x01, 0x00, 0x12, 0x34, 0x00, 0xAB}
	if _, ok := p.Parse(buf2); !ok {
		t.Fatal("expected accept on report ID match")
	}
}

func TestParserShortBuffer(t *testing.T) {
	p := Parser{ScratchIdx: 10, BtnStart: 0, BtnEnd: 2}
	buf := []byte{0xAA, 0xBB}
	if _, ok := p.Parse(buf); ok {
		t.Fatal("expected drop for short buffer (scratch idx out of range)")
	}
}
