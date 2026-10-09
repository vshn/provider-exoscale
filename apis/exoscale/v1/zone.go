package v1

// Zone is the datacenter identifier in which the instance runs in.
type Zone string

func (z Zone) String() string {
	return string(z)
}
