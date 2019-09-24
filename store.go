package namespace

// Store represents a pesistent store for managing and keeping alive namespaces
type Store interface {
	// Add will dup and save the namespace in the store
	Add(ns *Namespace, name string) error
	// Delete will close the namespace file and remove it from store
	Delete(typ Type, name string) error
	// Exists will check if a namespace with given type and name exists in the store
	Exists(typ Type, name string) bool
	// Get will dup and return the namespace with given type and name from store
	Get(typ Type, name string) (*Namespace, error)
	// List will return the names of saved namespaces for the given type
	List(typ Type) []string
}
