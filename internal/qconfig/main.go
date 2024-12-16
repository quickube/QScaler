package qconfig

import (
	"sync"
)

var (
	SecretToQConfigsRegistry = make(map[string][]string)
	RegistryMutex            sync.Mutex
)

func ListQConfigs(secretName string) []string {
	if _, ok := SecretToQConfigsRegistry[secretName]; !ok {
		return make([]string, 0)
	}
	return SecretToQConfigsRegistry[secretName]

}

func AddSecret(configName string, secretName string) {
	RegistryMutex.Lock()
	defer RegistryMutex.Unlock()

	SecretToQConfigsRegistry[secretName] = append(SecretToQConfigsRegistry[secretName], configName)
}

func RemoveSecret(secretName string) {
	RegistryMutex.Lock()
	defer RegistryMutex.Unlock()

	delete(SecretToQConfigsRegistry, secretName)
}
