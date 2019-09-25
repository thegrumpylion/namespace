package namespace

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

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
	// INVALID for use in TypeFromString
	INVALID = 0
)

var typeNameMap = map[Type]string{
	MNT:    "MNT",
	NET:    "NET",
	PID:    "PID",
	IPC:    "IPC",
	UTS:    "UTS",
	USER:   "USER",
	CGROUP: "CGROUP",
}

// String returns the uper case type of namespace
func (t Type) String() string {
	if s, ok := typeNameMap[t]; ok {
		return s
	}
	return ""
}

// StringLower returns lower case namspace type
func (t Type) StringLower() string {
	return strings.ToLower(t.String())
}

// TypeFromString returns a namespace type from a string. case insensitive
func TypeFromString(s string) Type {
	for t, n := range typeNameMap {
		if strings.ToUpper(s) == n {
			return t
		}
	}
	return INVALID
}

// Types returns a slice with all namespace types
func Types() []Type {
	out := []Type{}
	for t := range typeNameMap {
		out = append(out, t)
	}
	return out
}

// Mask is the value of ORed namespace types
type Mask int

// Has is true if mask is ORed with provided type
func (m Mask) Has(t Type) bool {
	return m&Mask(t) != 0
}

// Set adds namespace t to the mask and returns it
func (m Mask) Set(t Type) Mask {
	return m | Mask(t)
}

// Namespace represents an open file that points to some type of namspace
type Namespace struct {
	typ    Type
	file   *os.File
	stat   *syscall.Stat_t
	closed bool
}

// Type returns the namespace type
func (ns *Namespace) Type() Type {
	return ns.typ
}

// Fd returns the number of the file descriptor
func (ns *Namespace) Fd() int {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	return int(ns.file.Fd())
}

// Ino returns the inode number of namspace
func (ns *Namespace) Ino() uint64 {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	return ns.stat.Ino
}

type dev struct {
	Major uint32
	Minor uint32
}

// Dev returns the uint64 dev representation
func (d dev) Dev() uint64 {
	return unix.Mkdev(d.Major, d.Minor)
}

// Dev returns the inode number of namspace
func (ns *Namespace) Dev() dev {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	return dev{
		Major: unix.Major(ns.stat.Dev),
		Minor: unix.Minor(ns.stat.Dev),
	}
}

// FileName returns the name of file
func (ns *Namespace) FileName() string {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	return ns.file.Name()
}

// Set the callers namespace to ns
func (ns *Namespace) Set() error {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	return unix.Setns(ns.Fd(), int(ns.typ))
}

// Close the file descriptor holding the namespace
func (ns *Namespace) Close() error {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	ns.closed = true
	return ns.file.Close()
}

// SetAndClose sets the callers namespace to ns then closes the file
func (ns *Namespace) SetAndClose() error {
	err := unix.Setns(ns.Fd(), int(ns.typ))
	if err != nil {
		return err
	}
	return ns.Close()
}

// OwningUserNS returns the owning user namespace for a namespace
func (ns *Namespace) OwningUserNS() (*Namespace, error) {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	f, err := ioctlGetHierarchichal(ns.file.Fd(), unix.NS_GET_USERNS)
	if err != nil {
		return nil, err
	}
	stat, err := stat(f)
	if err != nil {
		return nil, err
	}
	return &Namespace{
		typ:  USER,
		file: f,
		stat: stat,
	}, nil
}

// Parent returns the parent namespace for a user or pid namespace
func (ns *Namespace) Parent() (*Namespace, error) {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	if !(ns.typ == PID || ns.typ == USER) {
		return nil, ErrNonHierarchicalNS
	}
	f, err := ioctlGetHierarchichal(ns.file.Fd(), unix.NS_GET_PARENT)
	if err != nil {
		return nil, err
	}
	stat, err := stat(f)
	if err != nil {
		return nil, err
	}
	return &Namespace{
		typ:  ns.typ,
		file: f,
		stat: stat,
	}, nil
}

// OwnerUID returns the owner UID for a user namespace
func (ns *Namespace) OwnerUID() (int, error) {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	if ns.typ != USER {
		return 0, ErrNonUserNS
	}
	return unix.IoctlGetInt(int(ns.file.Fd()), unix.NS_GET_OWNER_UID)
}

// Dup will return a duplicate of ns
func (ns *Namespace) Dup() (*Namespace, error) {
	if ns.closed {
		panic("acting on a closed namespace")
	}
	fd, err := unix.Dup(int(ns.file.Fd()))
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), ns.file.Name())
	// xxx: is stating here really necessary?
	stat, err := stat(f)
	if err != nil {
		return nil, err
	}
	return &Namespace{
		typ:  ns.typ,
		file: f,
		stat: stat,
	}, nil

}

// Open return a new namspace from the given path. It fails if the file doesn't point to a namespace
func Open(path string) (*Namespace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	t, err := ioctlGetType(f.Fd())
	if err != nil {
		return nil, ErrFileNotNamspace
	}
	stat, err := stat(f)
	if err != nil {
		return nil, err
	}
	return &Namespace{
		typ:  Type(t),
		file: f,
		stat: stat,
	}, nil
}

// OpenPID return a new namspace for a PID and Type. Needs procfs.
func OpenPID(pid int, t Type) (*Namespace, error) {
	return Open(filepath.Join(PROCFSPath, strconv.Itoa(pid), "ns", t.StringLower()))
}

// OpenSelf return a new namspace of type t of the caller. Needs procfs.
func OpenSelf(t Type) (*Namespace, error) {
	return Open(filepath.Join(PROCFSPath, "self", "ns", t.StringLower()))
}

func stat(f *os.File) (*syscall.Stat_t, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("stat not Stat_t")
	}
	return stat, nil
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
