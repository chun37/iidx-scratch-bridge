package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/chun37/iidx-scratch-bridge/internal/keyout"
)

// Config is the top-level user configuration.
type Config struct {
	Device  DeviceConfig      `toml:"device"`
	Scratch ScratchConfig     `toml:"scratch"`
	Buttons map[string]string `toml:"buttons"`
}

// DeviceConfig describes how to find and parse the HID device.
//
// VID and PID identify the USB device. ScratchAxisByteIndex is the
// byte offset of the scratch wheel position inside the Input Report,
// and ButtonsByteRange is the half-open [start, end) range of bytes
// holding the button bitmap.
//
// ReportID is optional: zero means "treat the first byte as the
// payload"; non-zero values mean "skip reports whose first byte does
// not match this report ID, and treat bytes after it as the payload."
type DeviceConfig struct {
	VID                  uint16 `toml:"vid"`
	PID                  uint16 `toml:"pid"`
	ScratchAxisByteIndex int    `toml:"scratch_axis_byte_index"`
	ButtonsByteRange     [2]int `toml:"buttons_byte_range"`
	ReportID             int    `toml:"report_id"`
}

// ScratchConfig tunes the scratch-to-pulse conversion.
type ScratchConfig struct {
	Threshold       int    `toml:"threshold"`
	PulseDurationMs int    `toml:"pulse_duration_ms"`
	UpKey           string `toml:"up_key"`
	DownKey         string `toml:"down_key"`
}

// Resolved is Config after parsing key names into VK codes and after
// defaults have been filled in.
type Resolved struct {
	Device      DeviceConfig
	Threshold   int
	PulseMs     int
	UpVK        keyout.VKCode
	DownVK      keyout.VKCode
	ButtonVKMap [16]keyout.VKCode // index is the bit position; 0 = unmapped
}

// Load reads and validates a TOML config file from path.
func Load(path string) (*Resolved, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return resolve(&cfg)
}

func resolve(cfg *Config) (*Resolved, error) {
	if cfg.Device.VID == 0 || cfg.Device.PID == 0 {
		return nil, fmt.Errorf("device.vid and device.pid are required")
	}
	if cfg.Device.ScratchAxisByteIndex < 0 {
		return nil, fmt.Errorf("device.scratch_axis_byte_index must be >= 0")
	}
	if cfg.Device.ButtonsByteRange[0] < 0 || cfg.Device.ButtonsByteRange[1] <= cfg.Device.ButtonsByteRange[0] {
		return nil, fmt.Errorf("device.buttons_byte_range must be [start, end) with end > start")
	}
	if cfg.Device.ButtonsByteRange[1]-cfg.Device.ButtonsByteRange[0] > 2 {
		return nil, fmt.Errorf("device.buttons_byte_range must span at most 2 bytes (16 bits)")
	}

	threshold := cfg.Scratch.Threshold
	if threshold <= 0 {
		threshold = 2
	}
	pulseMs := cfg.Scratch.PulseDurationMs
	if pulseMs <= 0 {
		pulseMs = 133
	}

	upName := cfg.Scratch.UpKey
	if upName == "" {
		upName = "UP"
	}
	downName := cfg.Scratch.DownKey
	if downName == "" {
		downName = "DOWN"
	}

	upVK, err := keyout.ParseKeyName(upName)
	if err != nil {
		return nil, fmt.Errorf("scratch.up_key: %w", err)
	}
	downVK, err := keyout.ParseKeyName(downName)
	if err != nil {
		return nil, fmt.Errorf("scratch.down_key: %w", err)
	}

	res := &Resolved{
		Device:    cfg.Device,
		Threshold: threshold,
		PulseMs:   pulseMs,
		UpVK:      upVK,
		DownVK:    downVK,
	}

	for name, keyName := range cfg.Buttons {
		idx, err := parseButtonName(name)
		if err != nil {
			return nil, fmt.Errorf("buttons.%s: %w", name, err)
		}
		vk, err := keyout.ParseKeyName(keyName)
		if err != nil {
			return nil, fmt.Errorf("buttons.%s: %w", name, err)
		}
		res.ButtonVKMap[idx] = vk
	}
	return res, nil
}

// parseButtonName accepts "b0".."b15" and returns the bit index.
func parseButtonName(name string) (int, error) {
	if len(name) < 2 || (name[0] != 'b' && name[0] != 'B') {
		return 0, fmt.Errorf("button name must look like b0..b15")
	}
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(name, "b"), "B"))
	if err != nil {
		return 0, fmt.Errorf("invalid button index: %w", err)
	}
	if n < 0 || n > 15 {
		return 0, fmt.Errorf("button index out of range: %d", n)
	}
	return n, nil
}
