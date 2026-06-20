//go:build !windows

package keyout

import (
	"fmt"
	"os"
)

// WinSender is a non-functional stub on non-Windows builds. It exists
// so cmd/scratch-bridge can be built on Linux for cross-compilation
// development. Calling Send only logs to stderr.
type WinSender struct{}

func NewWinSender() *WinSender { return &WinSender{} }

func (WinSender) Send(vk uint16, down bool) error {
	verb := "release"
	if down {
		verb = "press  "
	}
	fmt.Fprintf(os.Stderr, "[keyout-stub] %s vk=0x%02X\n", verb, vk)
	return nil
}
