package namespace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// PROCFSPath proc fs path
var PROCFSPath = "/proc"

// ErrFileNotNamspace returned when trying to open a file that is not a reference to some namspace
var ErrFileNotNamspace = errors.New("file not a namespace")

// ErrKernelNoSupport returned when OwningUserNS & Parent if the kernel does not support them
var ErrKernelNoSupport = errors.New("kernel does not support the opperation")

// ErrNotPermitted returned when requested namespace is outside of the caller's namespace scope or
// when attempting to obtain the parent of the initial user or PID namespace
var ErrNotPermitted = errors.New("operation not permitted")

// ErrNonHierarchicalNS returned when calling Parent from ns that is not pid or user
var ErrNonHierarchicalNS = errors.New("ns not hierarchical (pid or user)")

// ErrNonUserNS returned when calling OwnerUID on a non user namespace
var ErrNonUserNS = errors.New("only valid for user ns")

// Type of the namespace
type Type int

const (
	// MNT Mount namespace
	MNT Type = unix.CLONE_NEWNS
	// NET Network namespace
	NET = unix.CLONE_NEWNET
	// PID Process namespace
	PID = unix.CLONE_NEWPID
	// IPC Network namespace
	IPC = unix.CLONE_NEWIPC
	// UTS namespace
	UTS = unix.CLONE_NEWUTS
	// USER namespace
	USER = unix.CLONE_NEWUSER
	// CGROUP namespace
	CGROUP = unix.CLONE_NEWCGROUP
)

func (t Type) String() string {
	switch t {
	case MNT:
		return "MNT"
	case NET:
		return "NET"
	case PID:
		return "PID"
	case IPC:
		return "IPC"
	case UTS:
		return "UTS"
	case USER:
		return "USER"
	case CGROUP:
		return "CGROUP"
	default:
		return fmt.Sprintf("namespace.Type(%d)", t)
	}
}

// StringLower returns lower case namspace type
func (t Type) StringLower() string {
	return strings.ToLower(t.String())
}

// Namespace represents an open file that points to some type of namspace
type Namespace struct {
	File *os.File
	Type Type
}

// Set the callers namespace to ns
func (ns *Namespace) Set() error {
	return unix.Setns(int(ns.File.Fd()), int(ns.Type))
}

// OwningUserNS returns the owning user namespace for a namespace
func (ns *Namespace) OwningUserNS() (*Namespace, error) {
	f, err := ioctlGetHierarchichal(ns.File.Fd(), unix.NS_GET_USERNS)
	if err != nil {
		return nil, err
	}
	return &Namespace{f, USER}, nil
}

// Parent returns the parent namespace for a user or pid namespace
func (ns *Namespace) Parent() (*Namespace, error) {
	if !(ns.Type == PID || ns.Type == USER) {
		return nil, ErrNonHierarchicalNS
	}
	f, err := ioctlGetHierarchichal(ns.File.Fd(), unix.NS_GET_PARENT)
	if err != nil {
		return nil, err
	}
	return &Namespace{f, ns.Type}, nil
}

// OwnerUID returns the owner UID for a user namespace
func (ns *Namespace) OwnerUID() (int, error) {
	if ns.Type != USER {
		return 0, ErrNonUserNS
	}
	return unix.IoctlGetInt(int(ns.File.Fd()), unix.NS_GET_OWNER_UID)
}

// Open return a new namspace from the given path. It fails if the file doesn't point to a namespace
func Open(path string) (*Namespace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	t, err := ioctlGetType(f.Fd())
	if err != nil {
		return nil, err
	}
	return &Namespace{f, Type(t)}, nil
}

// OpenPID return a new namspace for a PID and Type. Needs procfs.
func OpenPID(pid int, t Type) (*Namespace, error) {
	return Open(filepath.Join(PROCFSPath, strconv.Itoa(pid), "ns", t.StringLower()))
}

// OpenSelf return a new namspace of type t of the caller. Needs procfs.
func OpenSelf(t Type) (*Namespace, error) {
	return Open(filepath.Join(PROCFSPath, "self", "ns", t.StringLower()))
}

func ioctlGetType(fd uintptr) (Type, error) {
	a, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.NS_GET_NSTYPE, uintptr(0))
	if e != 0 {
		if e == unix.ENOTTY {
			return Type(0), ErrFileNotNamspace
		}
		return Type(0), e
	}
	return Type(a), nil
}

func ioctlGetHierarchichal(fd, call uintptr) (*os.File, error) {
	fdOut, _, e := unix.Syscall(unix.SYS_IOCTL, fd, call, uintptr(0))
	if e != 0 {
		if e == unix.ENOTTY {
			return nil, ErrKernelNoSupport
		}
		if e == unix.EPERM {
			return nil, ErrNotPermitted
		}
		return nil, e
	}
	return os.NewFile(fdOut, ""), nil
}
