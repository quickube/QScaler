package qconfig

import (
	"slices"
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
	if _, ok := SecretToQConfigsRegistry[secretName]; !ok {
		SecretToQConfigsRegistry[secretName] = []string{configName}
		return
	}
	if ok := slices.Contains(SecretToQConfigsRegistry[secretName], configName); !ok {
		SecretToQConfigsRegistry[secretName] = append(SecretToQConfigsRegistry[secretName], configName)
	}
}

func PopSecret(secretName string) []string {
	RegistryMutex.Lock()
	defer RegistryMutex.Unlock()

	if qConfigs, ok := SecretToQConfigsRegistry[secretName]; !ok {
		return make([]string, 0)
	} else {
		delete(SecretToQConfigsRegistry, secretName)
		return qConfigs
	}
}
