//go:build windows

package keyout

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Windows SendInput definitions.
// Mirror of the C structs in WinUser.h. INPUT is a tagged union whose
// largest member is MOUSEINPUT (24 bytes on x64), so we model it with
// a fixed-size buffer big enough to hold any variant and lay the
// keyboard fields on top of the buffer with the right offsets.
//
// References:
//   https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput
//   https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-input

const (
	inputKeyboard   = 1
	keyeventfKeyUp  = 0x0002
	keyeventfScancode = 0x0008
)

// keyboardInput matches the KEYBDINPUT struct.
type keyboardInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// input matches the INPUT union. The union body is 32 bytes on x64
// (5 * uintptr inside MOUSEINPUT plus padding). Using a larger array
// here is safe because SendInput honours the cbSize argument we pass.
type input struct {
	Type uint32
	_    uint32 // padding so the union starts on an 8-byte boundary on x64
	Ki   keyboardInput
	_    [8]byte // padding so the struct matches sizeof(INPUT) on x64
}

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

// WinSender drives Windows SendInput.
type WinSender struct{}

// NewWinSender returns a ready-to-use Sender. The function is named to
// avoid clashing with the non-Windows stub in sender_other.go.
func NewWinSender() *WinSender { return &WinSender{} }

// Send delivers one keyboard event.
func (WinSender) Send(vk uint16, down bool) error {
	in := input{Type: inputKeyboard}
	in.Ki.WVk = vk
	if !down {
		in.Ki.DwFlags = keyeventfKeyUp
	}
	size := unsafe.Sizeof(in)
	ret, _, err := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&in)),
		size,
	)
	if ret != 1 {
		return fmt.Errorf("SendInput returned %d: %w", ret, err)
	}
	return nil
}
