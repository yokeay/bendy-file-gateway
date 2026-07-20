package storage

import "fmt"

var drivers = map[string]Factory{}

// Register registers a driver factory.
func Register(name string, factory Factory) {
	drivers[name] = factory
}

// Create creates a Driver instance from name and configuration.
func Create(name string, cfg map[string]string) (Driver, error) {
	factory, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
	return factory(cfg)
}

// ListDrivers returns all registered driver names.
func ListDrivers() []string {
	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	return names
}
