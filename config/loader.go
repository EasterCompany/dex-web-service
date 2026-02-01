package config

import (
	"fmt"

	sharedConfig "github.com/EasterCompany/dex-go-utils/config"
)

// LoadServiceMap loads the service-map.json file using the shared utility.
func LoadServiceMap() (*sharedConfig.ServiceMapConfig, error) {
	return sharedConfig.LoadServiceMap()
}

// LoadOptions loads the options.json file using the shared utility.
func LoadOptions() (*sharedConfig.OptionsConfig, error) {
	return sharedConfig.LoadOptions()
}

// LoadSystem loads the system.json file using the shared utility.
func LoadSystem() (*sharedConfig.SystemConfig, error) {
	return sharedConfig.LoadSystem()
}

// ResolveServiceHost finds a service by its ID and returns its "domain:port" or "domain" if no port.
func ResolveServiceHost(id string) (string, error) {
	sm, err := sharedConfig.LoadServiceMap()
	if err != nil {
		return "", err
	}

	svc, err := sm.ResolveService(id)
	if err != nil {
		return "", err
	}

	host := svc.Domain
	if host == "" {
		host = "127.0.0.1"
	}
	if svc.Port != "" {
		host = fmt.Sprintf("%s:%s", host, svc.Port)
	}
	return host, nil
}
