//go:build !darwin

package main

import (
	"syscall"
	"time"
)

// ponytail: PID reuse after a reboot can show a dead session as open; add a /proc start-time check if it bites.
func alive(pid int, _ time.Time) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
