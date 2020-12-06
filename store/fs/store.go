package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/thegrumpylion/namespace"
	"github.com/thegrumpylion/namespace/store"
	"golang.org/x/sys/unix"
)

type fsStore struct {
	sync.RWMutex
	root string
	flat bool
}

// FsType is the type of FsStore.
type FsType uint8

const (
	// FsNone use the filesystem as is
	FsNone FsType = iota
	// FsTmpfs use tempfs on store dir for FsStore
	FsTmpfs
	// FsBind bind mount the store dir for FsStore
	FsBind
)

// NewFsStore returns a new namespace fs store
func NewFsStore(root string, ft FsType, flat bool) (store.Store, error) {
	root = filepath.Clean(root)
	switch ft {
	case FsTmpfs:
		if err := unix.Mount("tmpfs", root, "tmpfs", 0, ""); err != nil {
			return nil, fmt.Errorf("tmpfs mount %s fail: %v", root, err)
		}
		if err := unix.Mount("tmpfs", root, "tmpfs", unix.MS_PRIVATE, ""); err != nil {
			return nil, fmt.Errorf("tmpfs make private %s fail: %v", root, err)
		}
	case FsBind:
		if err := unix.Mount(root, root, "", unix.MS_BIND, ""); err != nil {
			return nil, fmt.Errorf("bind mount %s fail: %v", root, err)
		}
		if err := unix.Mount("", root, "", unix.MS_PRIVATE, ""); err != nil {
			return nil, fmt.Errorf("bind mount make private %s fail: %v", root, err)
		}
	}
	empty, err := dirIsEmpty(root)
	if err != nil {
		return nil, err
	}
	if empty && !flat {
		for _, t := range namespace.Types() {
			err := os.Mkdir(filepath.Join(root, t.StringLower()), 0666)
			if err != nil {
				return nil, err
			}
		}
	}
	return &fsStore{
		root: root,
		flat: flat,
	}, nil
}

// Add bind mounts the namespace in the fs store
func (s *fsStore) Add(ns *namespace.Namespace, name string) error {
	if _, err := os.Stat(ns.FileName()); err != nil {
		return err
	}
	src := ns.FileName()

	trgt := s.targetPath(name, ns.Type())

	if _, err := os.Stat(trgt); err == nil {
		return store.ErrExists
	}

	f, err := os.Create(trgt)
	if err != nil {
		return err
	}
	defer f.Close()

	return unix.Mount(src, trgt, "", unix.MS_BIND, "")
}

// Delete closse the namespace file and removes it from store
func (s *fsStore) Delete(typ namespace.Type, name string) error {
	if !s.Exists(typ, name) {
		return store.ErrNotExists
	}
	trgt := s.targetPath(name, typ)
	err := unix.Unmount(trgt, 0)
	if err != nil {
		return err
	}
	return os.Remove(trgt)
}

// Exists checks if a namespace with given type and name exists in the store
func (s *fsStore) Exists(typ namespace.Type, name string) bool {
	trgt := s.targetPath(name, typ)
	if _, err := os.Stat(trgt); err == nil {
		return true
	}
	return false
}

// Get dups and returns the namespace with given type and name from store
func (s *fsStore) Get(typ namespace.Type, name string) (*namespace.Namespace, error) {
	if !s.Exists(typ, name) {
		return nil, store.ErrNotExists
	}
	trgt := s.targetPath(name, typ)
	return namespace.FromPath(trgt)
}

// List returns the names of saved namespaces for the given type
func (s *fsStore) List(typ namespace.Type) []string {
	out := []string{}
	dir := filepath.Join(s.root, typ.StringLower())
	fl, err := ioutil.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, inf := range fl {
		out = append(out, inf.Name())
	}
	return out
}

func (s *fsStore) targetPath(name string, typ namespace.Type) string {
	if s.flat {
		return filepath.Join(s.root, name)
	}
	return filepath.Join(s.root, typ.StringLower(), name)
}

func dirIsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
