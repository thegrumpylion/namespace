package namespace

import (
	"testing"

	"golang.org/x/sys/unix"
)

func testStore(t *testing.T, s Store, pfx string) {

	m := NewMask().SetAll()

	c, err := newProcess(m)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Wait()

	ppid := c.Process.Pid

	nsname := func(nst Type) string {
		return pfx + nst.StringLower()
	}

	for _, nsType := range Types() {
		ns, err := FromPID(ppid, nsType)
		if err != nil {
			t.Fatalf("fail to get %s ns for pid %d", nsType, ppid)
		}
		s.Add(ns, nsname(nsType))
	}

	for _, nsType := range Types() {
		ns, err := s.Get(nsType, nsname(nsType))
		if err != nil {
			t.Fatal("could not get", nsname(nsType))
		}
		if err := ns.Close(); err != nil {
			t.Fatal("fail to close", nsname(nsType))
		}
	}

	for _, nsType := range Types() {
		lst := s.List(nsType)
		if len(lst) != 1 {
			t.Fatal("list should only have one entry for namespace", nsType.String())
		}
		if lst[0] != nsname(nsType) {
			t.Fatalf("expecting %s but got %s for namespace %s", nsname(nsType), lst[0], nsType)
		}
	}

	for _, nsType := range Types() {
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

	s, err := NewFsStore(tmp, FsTmpfs)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Unmount(tmp, 0)

	testStore(t, s, "tmpfs_")
}

func TestFsStoreBind(t *testing.T) {
	tmp := t.TempDir()

	s, err := NewFsStore(tmp, FsBind)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Unmount(tmp, 0)

	testStore(t, s, "bind_")
}

func TestMemStore(t *testing.T) {

	s := NewMemStore()

	testStore(t, s, "mem_")
}
