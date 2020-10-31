package namespace

// Mask is the value of ORed namespace types
type Mask int

// NewMask returns a new Mask
func NewMask() Mask {
	return Mask(0)
}

// Has is true if mask is ORed with provided type
func (m Mask) Has(t Type) bool {
	return m&Mask(t) != 0
}

// Uintptr returns Mask as uintptr value
func (m Mask) Uintptr() uintptr {
	return uintptr(m)
}

// Set adds namespace t to the mask and returns it
func (m Mask) Set(t Type) Mask {
	return m | Mask(t)
}

// Remove removes namespace t from the mask and returns it
func (m Mask) Remove(t Type) Mask {
	return m & ^Mask(t)
}

// SetAll returns a mask with all namespaces set
func (m Mask) SetAll() Mask {
	return Mask(MNT | NET | PID | IPC | UTS | USER | CGROUP)
}
