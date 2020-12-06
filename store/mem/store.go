package mem

import (
	"sort"
	"sync"

	"github.com/thegrumpylion/namespace"
	"github.com/thegrumpylion/namespace/store"
)

type memStore struct {
	sync.RWMutex
	data map[namespace.Type]map[string]*namespace.Namespace
}

// NewMemStore returns a new namespace memory store
func NewMemStore() store.Store {
	s := &memStore{
		data: map[namespace.Type]map[string]*namespace.Namespace{},
	}
	for _, t := range namespace.Types() {
		s.data[t] = map[string]*namespace.Namespace{}
	}
	return s
}

// Add dups and saves the namespace in the store
func (s *memStore) Add(ns *namespace.Namespace, name string) error {
	newNs, err := ns.Dup()
	if err != nil {
		return err
	}
	if _, ok := s.data[ns.Type()][name]; ok {
		return store.ErrExists
	}
	s.data[ns.Type()][name] = newNs
	return nil
}

// Delete closse the namespace file and removes it from store
func (s *memStore) Delete(typ namespace.Type, name string) error {
	if _, ok := s.data[typ][name]; !ok {
		return store.ErrNotExists
	}
	// keep a ref to the ns
	ns := s.data[typ][name]
	delete(s.data[typ], name)
	// close
	return ns.Close()
}

// Exists checks if a namespace with given type and name exists in the store
func (s *memStore) Exists(typ namespace.Type, name string) bool {
	_, ok := s.data[typ][name]
	return ok
}

// Get dups and returns the namespace with given type and name from store
func (s *memStore) Get(typ namespace.Type, name string) (*namespace.Namespace, error) {
	if ns, ok := s.data[typ][name]; ok {
		newNs, err := ns.Dup()
		if err != nil {
			return nil, err
		}
		return newNs, nil
	}
	return nil, store.ErrNotExists
}

// List returns the names of saved namespaces for the given type
func (s *memStore) List(typ namespace.Type) []string {
	out := []string{}
	for s := range s.data[typ] {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
