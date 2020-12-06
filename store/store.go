package store

import (
	"errors"

	"github.com/thegrumpylion/namespace"
)

// Store represents a pesistent store for managing and keeping alive namespaces
type Store interface {
	// Add dups and saves the namespace in the store
	Add(ns *namespace.Namespace, name string) error
	// Delete closse the namespace file and removes it from store
	Delete(typ namespace.Type, name string) error
	// Exists checks if a namespace with given type and name exists in the store
	Exists(typ namespace.Type, name string) bool
	// Get dups and returns the namespace with given type and name from store
	Get(typ namespace.Type, name string) (*namespace.Namespace, error)
	// List returns the names of saved namespaces for the given type
	List(typ namespace.Type) []string
}

// ErrExists is returned when trying to add new namespace with existing name
var ErrExists = errors.New("namespace already in store")

// ErrNotExists is returned when trying to get a namespace with unknown name
var ErrNotExists = errors.New("namespace not in store")
