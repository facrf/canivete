package main

import (
	"os/exec"
	"syscall"
	"time"
)

func configureWorkerProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Include pdftoppm descendants in cancellation, not just the Go worker.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = time.Second
}
