package v1alpha1

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

// ReferencesSecret checks if any field in the ScalerConfig references a Secret with the given name.
func (sc *ScalerConfig) ReferencesSecret(secretName string) bool {
	return referencesSecret(reflect.ValueOf(sc), secretName)
}

// referencesSecret is a helper function that recursively checks if any field contains a SecretKeySelector with the specified name.
func referencesSecret(val reflect.Value, secretName string) bool {
	if !val.IsValid() {
		return false
	}

	// Dereference pointers and interfaces to get the underlying value.
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return false
		}
		val = val.Elem()
	}

	// Only process struct types.
	if val.Kind() != reflect.Struct {
		return false
	}

	// Iterate over the struct's fields.
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		// Check if the field is of type *corev1.SecretKeySelector.
		if field.Type() == reflect.TypeOf(&corev1.SecretKeySelector{}) {
			if field.IsNil() {
				continue
			}
			secretKeySelector := field.Interface().(*corev1.SecretKeySelector)
			if secretKeySelector.Name == secretName {
				return true
			}
		}

		// Recursively check embedded structs or pointers to structs.
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct) {
			if referencesSecret(field, secretName) {
				return true
			}
		}
	}

	return false
}
