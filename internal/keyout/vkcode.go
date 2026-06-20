package keyout

import (
	"fmt"
	"strings"
)

// VKCode is a Windows Virtual-Key code.
type VKCode uint16

// Selected subset of Windows Virtual-Key codes.
// https://learn.microsoft.com/en-us/windows/win32/inputdev/virtual-key-codes
const (
	VKBack       VKCode = 0x08
	VKTab        VKCode = 0x09
	VKReturn     VKCode = 0x0D
	VKShift      VKCode = 0x10
	VKControl    VKCode = 0x11
	VKMenu       VKCode = 0x12
	VKEscape     VKCode = 0x1B
	VKSpace      VKCode = 0x20
	VKPageUp     VKCode = 0x21
	VKPageDown   VKCode = 0x22
	VKEnd        VKCode = 0x23
	VKHome       VKCode = 0x24
	VKLeft       VKCode = 0x25
	VKUp         VKCode = 0x26
	VKRight      VKCode = 0x27
	VKDown       VKCode = 0x28
	VKInsert     VKCode = 0x2D
	VKDelete     VKCode = 0x2E
)

var nameToVK = map[string]VKCode{
	"BACK":       VKBack,
	"BACKSPACE":  VKBack,
	"TAB":        VKTab,
	"ENTER":      VKReturn,
	"RETURN":     VKReturn,
	"SHIFT":      VKShift,
	"CTRL":       VKControl,
	"CONTROL":    VKControl,
	"ALT":        VKMenu,
	"ESC":        VKEscape,
	"ESCAPE":     VKEscape,
	"SPACE":      VKSpace,
	"PAGEUP":     VKPageUp,
	"PAGEDOWN":   VKPageDown,
	"END":        VKEnd,
	"HOME":       VKHome,
	"LEFT":       VKLeft,
	"UP":         VKUp,
	"RIGHT":      VKRight,
	"DOWN":       VKDown,
	"INSERT":     VKInsert,
	"INS":        VKInsert,
	"DELETE":     VKDelete,
	"DEL":        VKDelete,
}

// ParseKeyName maps a human-friendly key name to a Windows VK code.
//
// Single letters A-Z map to 0x41-0x5A, single digits 0-9 to 0x30-0x39,
// "F1".."F24" map to the function-key range, and the names in nameToVK
// cover the rest. Names are case-insensitive.
func ParseKeyName(name string) (VKCode, error) {
	if name == "" {
		return 0, fmt.Errorf("empty key name")
	}
	upper := strings.ToUpper(strings.TrimSpace(name))

	if len(upper) == 1 {
		c := upper[0]
		switch {
		case c >= 'A' && c <= 'Z':
			return VKCode(c), nil
		case c >= '0' && c <= '9':
			return VKCode(c), nil
		}
	}

	if len(upper) >= 2 && upper[0] == 'F' {
		var n int
		if _, err := fmt.Sscanf(upper[1:], "%d", &n); err == nil {
			if n >= 1 && n <= 24 {
				return VKCode(0x6F + n), nil
			}
		}
	}

	if vk, ok := nameToVK[upper]; ok {
		return vk, nil
	}
	return 0, fmt.Errorf("unknown key name: %q", name)
}
