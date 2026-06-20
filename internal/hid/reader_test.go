package hid

import "testing"

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
