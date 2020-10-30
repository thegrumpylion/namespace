package namespace

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"golang.org/x/sys/unix"
)

// Store represents a pesistent store for managing and keeping alive namespaces
type Store interface {

	// Add dups and saves the namespace in the store
	Add(ns *Namespace, name string) error
	// Delete closse the namespace file and removes it from store
	Delete(typ Type, name string) error
	// Exists checks if a namespace with given type and name exists in the store
	Exists(typ Type, name string) bool
	// Get dups and returns the namespace with given type and name from store
	Get(typ Type, name string) (*Namespace, error)
	// List returns the names of saved namespaces for the given type
	List(typ Type) []string
}

// ErrExists is returned when trying to add new namespace with existing name
var ErrExists = errors.New("namespace already in store")

// ErrNotExists is returned when trying to get a namespace with unknown name
var ErrNotExists = errors.New("namespace not in store")

type memStore struct {
	sync.RWMutex
	data map[Type]map[string]*Namespace
}

// NewMemStore returns a new namespace memory store
func NewMemStore() Store {
	s := &memStore{
		data: map[Type]map[string]*Namespace{},
	}
	for _, t := range Types() {
		s.data[t] = map[string]*Namespace{}
	}
	return s
}

// Add dups and saves the namespace in the store
func (s *memStore) Add(ns *Namespace, name string) error {
	newNs, err := ns.Dup()
	if err != nil {
		return err
	}
	if _, ok := s.data[ns.Type()][name]; ok {
		return ErrExists
	}
	s.data[ns.Type()][name] = newNs
	return nil
}

// Delete closse the namespace file and removes it from store
func (s *memStore) Delete(typ Type, name string) error {
	if _, ok := s.data[typ][name]; !ok {
		return ErrNotExists
	}
	// keep a ref to the ns
	ns := s.data[typ][name]
	delete(s.data[typ], name)
	// close
	return ns.Close()
}

// Exists checks if a namespace with given type and name exists in the store
func (s *memStore) Exists(typ Type, name string) bool {
	_, ok := s.data[typ][name]
	return ok
}

// Get dups and returns the namespace with given type and name from store
func (s *memStore) Get(typ Type, name string) (*Namespace, error) {
	if ns, ok := s.data[typ][name]; ok {
		newNs, err := ns.Dup()
		if err != nil {
			return nil, err
		}
		return newNs, nil
	}
	return nil, ErrNotExists
}

// List returns the names of saved namespaces for the given type
func (s *memStore) List(typ Type) []string {
	out := []string{}
	for s := range s.data[typ] {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

type fsStore struct {
	sync.RWMutex
	root string
}

// FsType is the type of FsStore.
type FsType uint8

const (
	// FsTempfs use tempfs on store dir for FsStore
	FsTempfs FsType = iota
	// FsBind bind mount the store dir for FsStore
	FsBind
)

// NewFsStore returns a new namespace fs store
func NewFsStore(root string, ft FsType) (Store, error) {
	root = filepath.Clean(root)
	switch ft {
	case FsTempfs:
		if err := unix.Mount("tmpfs", root, "tmpfs", unix.MS_PRIVATE, ""); err != nil {
			return nil, err
		}
	case FsBind:
		if err := unix.Mount(root, root, "", unix.MS_PRIVATE|unix.MS_BIND, ""); err != nil {
			return nil, err
		}
	}
	for _, t := range Types() {
		err := os.Mkdir(filepath.Join(root, t.StringLower()), 0666)
		if err != nil {
			return nil, err
		}
	}
	return &fsStore{
		root: root,
	}, nil
}

// Add dups and saves the namespace in the store
func (s *fsStore) Add(ns *Namespace, name string) error {
	if _, err := os.Stat(ns.FileName()); err != nil {
		return err
	}

	src := ns.FileName()
	trgt := filepath.Join(s.root, ns.Type().StringLower(), name)

	if _, err := os.Stat(trgt); err == nil {
		return ErrExists
	}

	f, err := os.Create(trgt)
	if err != nil {
		return err
	}
	defer f.Close()

	return unix.Mount(src, trgt, "", unix.MS_BIND, "")
}

// Delete closse the namespace file and removes it from store
func (s *fsStore) Delete(typ Type, name string) error {
	trgt := filepath.Join(s.root, typ.StringLower(), name)
	if _, err := os.Stat(trgt); err != nil {
		return ErrNotExists
	}
	err := unix.Unmount(trgt, 0)
	if err != nil {
		return err
	}
	return os.Remove(trgt)
}

// Exists checks if a namespace with given type and name exists in the store
func (s *fsStore) Exists(typ Type, name string) bool {
	trgt := filepath.Join(s.root, typ.StringLower(), name)
	if _, err := os.Stat(trgt); err == nil {
		return true
	}
	return false
}

// Get dups and returns the namespace with given type and name from store
func (s *fsStore) Get(typ Type, name string) (*Namespace, error) {
	trgt := filepath.Join(s.root, typ.StringLower(), name)
	return Open(trgt)
}

// List returns the names of saved namespaces for the given type
func (s *fsStore) List(typ Type) []string {
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
