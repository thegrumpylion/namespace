package store_test

import (
	"os/exec"
	"syscall"
	"testing"

	"github.com/thegrumpylion/namespace"
	"github.com/thegrumpylion/namespace/store"
	"github.com/thegrumpylion/namespace/store/fs"
	"github.com/thegrumpylion/namespace/store/mem"
	"golang.org/x/sys/unix"
)

func newProcess(m namespace.Mask) (*exec.Cmd, error) {
	c := exec.Command("sleep", "7200")
	c.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: m.Uintptr(),
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	return c, nil
}

func testStore(t *testing.T, s store.Store, pfx string) {

	m := namespace.NewMask().SetAll()

	c, err := newProcess(m)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Wait()

	ppid := c.Process.Pid

	nsname := func(nst namespace.Type) string {
		return pfx + nst.StringLower()
	}

	for _, nsType := range namespace.Types() {
		ns, err := namespace.FromPID(ppid, nsType)
		if err != nil {
			t.Fatalf("fail to get %s ns for pid %d", nsType, ppid)
		}
		s.Add(ns, nsname(nsType))
	}

	for _, nsType := range namespace.Types() {
		ns, err := s.Get(nsType, nsname(nsType))
		if err != nil {
			t.Fatal("could not get", nsname(nsType))
		}
		if err := ns.Close(); err != nil {
			t.Fatal("fail to close", nsname(nsType))
		}
	}

	for _, nsType := range namespace.Types() {
		lst := s.List(nsType)
		if len(lst) != 1 {
			t.Fatal("list should only have one entry for namespace", nsType.String())
		}
		if lst[0] != nsname(nsType) {
			t.Fatalf("expecting %s but got %s for namespace %s", nsname(nsType), lst[0], nsType)
		}
	}

	for _, nsType := range namespace.Types() {
		if err := s.Delete(nsType, nsname(nsType)); err != nil {
			t.Fatal("fail to delete", nsname(nsType))
		}
	}

	if err = c.Process.Kill(); err != nil {
		t.Fatal("fail to kill process", ppid)
	}

}

func TestFsStoreTmpfs(t *testing.T) {
	tmp := t.TempDir()

	s, err := fs.NewFsStore(tmp, fs.FsTmpfs, false)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Unmount(tmp, 0)

	testStore(t, s, "tmpfs_")
}

func TestFsStoreBind(t *testing.T) {
	tmp := t.TempDir()

	s, err := fs.NewFsStore(tmp, fs.FsBind, false)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Unmount(tmp, 0)

	testStore(t, s, "bind_")
}

func TestMemStore(t *testing.T) {

	s := mem.NewMemStore()

	testStore(t, s, "mem_")
}
