package namespace

import (
	"os/exec"
	"syscall"
	"testing"
)

func newProcess(m Mask) (*exec.Cmd, error) {
	c := exec.Command("sleep", "7200")
	c.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: m.Uintptr(),
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	return c, nil
}

func TestNewProc(t *testing.T) {
	m := NewMask().SetAll()

	c, err := newProcess(m)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Wait()

	ppid := c.Process.Pid

	for _, nsType := range Types() {
		_, err := FromPID(ppid, nsType)
		if err != nil {
			t.Fatalf("fail to get %s ns for pid %d", nsType, ppid)
		}
	}

	if err = c.Process.Kill(); err != nil {
		t.Fatal("fail to kill process", ppid)
	}
}
