package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWorkerProcessHelper(t *testing.T) {
	mode := os.Getenv("CANIVETE_PROCESS_TEST")
	if mode == "child" {
		time.Sleep(time.Minute)
		return
	}
	if mode != "parent" {
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestWorkerProcessHelper$")
	cmd.Env = append(os.Environ(), "CANIVETE_PROCESS_TEST=child")
	if err := cmd.Start(); err != nil {
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("CANIVETE_CHILD_PID"), []byte(strconv.Itoa(cmd.Process.Pid)), 0600); err != nil {
		os.Exit(3)
	}
	_ = cmd.Wait()
}

func TestCancellationKillsWorkerDescendants(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWorkerProcessHelper$")
	cmd.Env = append(os.Environ(), "CANIVETE_PROCESS_TEST=parent", "CANIVETE_CHILD_PID="+pidFile)
	configureWorkerProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cancel(); _ = cmd.Wait() }()
	var child int
	for child == 0 {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			child, _ = strconv.Atoi(string(data))
		}
		if ctx.Err() != nil {
			t.Fatal("worker did not create descendant before deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := cmd.Wait(); err == nil {
		t.Fatal("cancelled worker exited successfully")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := syscall.Kill(child, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		// An orphan may await reaping by the host's PID 1 after its group is
		// killed. A zombie has terminated and consumes no processing resources.
		state, readErr := os.ReadFile("/proc/" + strconv.Itoa(child) + "/stat")
		if readErr == nil && strings.Contains(string(state), ") Z ") {
			return
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(child, syscall.SIGKILL)
			t.Fatalf("worker descendant %d survived cancellation", child)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
