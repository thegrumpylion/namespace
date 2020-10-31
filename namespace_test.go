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

func TestHierarchical(t *testing.T) {
	m := NewMask().SetAll()

	c, err := newProcess(m)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Wait()

	ppid := c.Process.Pid

	mh := NewMask().Set(PID).Set(USER)
	mnh := NewMask().SetAll().Remove(PID).Remove(USER)

	for _, nsType := range Types() {
		ns, err := FromPID(ppid, nsType)
		if err != nil {
			t.Fatalf("fail to get %s ns for pid %d: %v", nsType, ppid, err)
		}

		prnt := &Namespace{}

		if nsType == USER {
			prnt, err = ns.OwningUserNS()
		} else {
			prnt, err = ns.Parent()
		}

		if mnh.Has(ns.Type()) {
			if err == nil {
				t.Fatal("ns.Parent should have failed for", ns.Type().String())
			}
			if err != ErrNonHierarchicalNS {
				t.Fatal("error should have been ErrNonHierarchicalNS instead of", err)
			}
			continue
		}

		if err != nil {
			t.Fatal(nsType.StringLower(), err)
		}

		if mh.Has(ns.Type()) {
			slf, err := Self(ns.Type())
			if err != nil {
				t.Fatal("could not get Self", ns.Type().StringLower())
			}
			if slf.Ino() != prnt.Ino() {
				t.Fatal("self and parent Ino should have match", slf.Ino(), prnt.Ino())
			}
		}

	}

	if err = c.Process.Kill(); err != nil {
		t.Fatal("fail to kill process", ppid)
	}
}
