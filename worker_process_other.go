//go:build !linux

package main

import (
	"os/exec"
	"time"
)

func configureWorkerProcess(cmd *exec.Cmd) {
	cmd.WaitDelay = time.Second
}
