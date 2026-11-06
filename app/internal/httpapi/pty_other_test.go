//go:build !linux && !darwin

package httpapi

import (
	"errors"
	"os"
)

func openPTY() (*os.File, string, error) { return nil, "", errors.New("pty unsupported on this OS") }
