package naming

import (
	"errors"

	"github.com/JellyTony/goim"
)

var (
	ErrNotFound = errors.New("service no found")
)

// Naming defined methods of the naming service
type Naming interface {
	Find(serviceName string, tags ...string) ([]goim.ServiceRegistration, error)
	Subscribe(serviceName string, callback func(services []goim.ServiceRegistration)) error
	Unsubscribe(serviceName string) error
	Register(service goim.ServiceRegistration) error
	Deregister(serviceID string) error
}
