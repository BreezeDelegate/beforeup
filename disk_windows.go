package main

import "errors"

// Windows host disk space does not describe Docker Desktop VM storage.
func freeBytes(string) (uint64, error) {
	return 0, errors.New("Docker Desktop VM disk space cannot be inspected from Windows")
}
