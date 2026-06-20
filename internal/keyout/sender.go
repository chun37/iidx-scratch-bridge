package keyout

// Sender delivers a key press or release to the OS.
type Sender interface {
	Send(vk uint16, down bool) error
}
