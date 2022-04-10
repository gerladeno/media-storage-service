package common

import (
	"errors"
	"os"
)

func RunsInContainer() bool {
	_, err := os.Stat("/.dockerenv")
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	panic(err)
}
