package namespace

import "testing"

func TestMask(t *testing.T) {
	{
		m := NewMask().SetAll()

		if m != Mask(MNT|NET|PID|IPC|UTS|USER|CGROUP) {
			t.Fatal("mask is not Mask(MNT | NET | PID | IPC | UTS | USER | CGROUP)")
		}
	}
	{
		m := NewMask().SetAll().
			Remove(MNT)

		if m != Mask(NET|PID|IPC|UTS|USER|CGROUP) {
			t.Fatal("mask is not Mask(NET | PID | IPC | UTS | USER | CGROUP)")
		}
	}
	{
		m := NewMask().SetAll().
			Remove(MNT).
			Remove(CGROUP)

		if m != Mask(NET|PID|IPC|UTS|USER) {
			t.Fatal("mask is not Mask(NET | PID | IPC | UTS | USER)")
		}
	}
	{
		m := NewMask().
			Set(PID).
			Set(NET).
			Set(UTS)

		if m != Mask(NET|PID|UTS) {
			t.Fatal("mask is not Mask(NET | PID | UTS)")
		}
	}
	{
		m := NewMask().
			Set(MNT).
			Set(USER).
			Set(CGROUP)

		if !(m.Has(MNT) && m.Has(USER) && m.Has(CGROUP)) {
			t.Fatal("mask is not Mask(MNT | USER | CGROUP)")
		}

		if m.Has(NET) || m.Has(PID) || m.Has(UTS) || m.Has(IPC) {
			t.Fatal("mask should not have NET or PID or UTS or IPC set")
		}
	}
}
