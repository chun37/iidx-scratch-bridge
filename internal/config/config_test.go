package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chun37/iidx-scratch-bridge/internal/keyout"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDefaults(t *testing.T) {
	body := `
[device]
vid = 0x1ccf
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [2, 4]

[scratch]
`
	p := writeTemp(t, body)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Threshold != 2 {
		t.Errorf("default threshold = %d, want 2", res.Threshold)
	}
	if res.PulseMs != 133 {
		t.Errorf("default pulse = %d, want 133", res.PulseMs)
	}
	if res.UpVK != keyout.VKUp || res.DownVK != keyout.VKDown {
		t.Errorf("default keys wrong: up=%#x down=%#x", res.UpVK, res.DownVK)
	}
}

func TestLoadButtons(t *testing.T) {
	body := `
[device]
vid = 0x1ccf
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [2, 4]

[buttons]
b0 = "Z"
b1 = "S"
b7 = "ENTER"
b15 = "BACKSPACE"
`
	p := writeTemp(t, body)
	res, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.ButtonVKMap[0] != 0x5A {
		t.Errorf("b0=%#x, want Z(0x5A)", res.ButtonVKMap[0])
	}
	if res.ButtonVKMap[7] != keyout.VKReturn {
		t.Errorf("b7=%#x, want ENTER", res.ButtonVKMap[7])
	}
	if res.ButtonVKMap[15] != keyout.VKBack {
		t.Errorf("b15=%#x, want BACKSPACE", res.ButtonVKMap[15])
	}
	if res.ButtonVKMap[2] != 0 {
		t.Errorf("b2 should be unmapped, got %#x", res.ButtonVKMap[2])
	}
}

func TestLoadValidation(t *testing.T) {
	cases := map[string]string{
		"no vid": `
[device]
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [2, 4]
`,
		"bad buttons range": `
[device]
vid = 0x1ccf
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [4, 4]
`,
		"unknown key": `
[device]
vid = 0x1ccf
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [2, 4]

[scratch]
up_key = "NOPE"
`,
		"bad button index": `
[device]
vid = 0x1ccf
pid = 0x8048
scratch_axis_byte_index = 5
buttons_byte_range = [2, 4]

[buttons]
b99 = "A"
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			p := writeTemp(t, body)
			if _, err := Load(p); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
