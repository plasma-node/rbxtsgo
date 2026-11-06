package watch

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestInterruptExitsCleanly sends the real binary what Ctrl-C and Ctrl-Break send. It has to stop and exit 0.
func TestInterruptExitsCleanly(t *testing.T) {
	state := baseState()
	_, live, _ := newDirs(t, "interrupt")
	writeState(t, live, render(state), nil)
	cmd := exec.Command(testBinary(t), "-w", "-p", live)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	ready := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Watching for file changes.") {
				select {
				case ready <- true:
				default:
				}
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(time.Minute):
		t.Fatal("no first build")
	}
	const ctrlBreak = 1
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	sent, _, callErr := kernel32.NewProc("GenerateConsoleCtrlEvent").Call(ctrlBreak, uintptr(cmd.Process.Pid))
	if sent == 0 {
		t.Skipf("this process has no console to send the event through: %v", callErr)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the watcher should exit 0: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the watcher didn't stop")
	}
}
