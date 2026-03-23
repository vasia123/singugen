package selfupdate

import (
	"fmt"
	"os"
	"syscall"
)

// SignalParentRestart sends SIGUSR1 to the parent process (supervisor)
// to trigger a child restart with updated code.
func SignalParentRestart() error {
	ppid := os.Getppid()
	if err := validateParentPID(ppid); err != nil {
		return err
	}
	return syscall.Kill(ppid, syscall.SIGUSR1)
}

func validateParentPID(ppid int) error {
	if ppid <= 1 {
		return fmt.Errorf("selfupdate: no parent supervisor (ppid=%d)", ppid)
	}
	return nil
}
