package main

import (
	"time"

	"golang.org/x/sys/unix"
)

// A reused PID has a different start time than the registry's startedAt.
func alive(pid int, started time.Time) bool {
	k, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || k.Proc.P_pid != int32(pid) {
		return false
	}
	ps := time.Unix(int64(k.Proc.P_starttime.Sec), 0)
	d := ps.Sub(started)
	return d > -time.Minute && d < time.Minute
}
