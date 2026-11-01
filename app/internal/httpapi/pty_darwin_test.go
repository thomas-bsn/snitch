//go:build darwin

package httpapi

import (
	"bytes"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// openPTY opens a pseudo-terminal pair without any external interpreter.
func openPTY() (master *os.File, slavePath string, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, "", err
	}
	fd := master.Fd()
	ioctl := func(req uintptr, arg unsafe.Pointer) error {
		if _, _, e := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
			return e
		}
		return nil
	}
	name := make([]byte, 128)
	if err = ioctl(unix.TIOCPTYGRANT, nil); err == nil {
		if err = ioctl(unix.TIOCPTYUNLK, nil); err == nil {
			err = ioctl(unix.TIOCPTYGNAME, unsafe.Pointer(&name[0]))
		}
	}
	if err != nil {
		master.Close()
		return nil, "", err
	}
	return master, string(name[:bytes.IndexByte(name, 0)]), nil
}
