//go:build linux

package httpapi

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// openPTY opens a pseudo-terminal pair without any external interpreter.
func openPTY() (master *os.File, slavePath string, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, "", err
	}
	if err = unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		master.Close()
		return nil, "", err
	}
	n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		master.Close()
		return nil, "", err
	}
	return master, fmt.Sprintf("/dev/pts/%d", n), nil
}
