package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

// GeneratePodTemplateHash generates a deterministic hash for a pod template
func GeneratePodTemplateHash(podSpec corev1.PodSpec) (string, error) {
	// Serialize the pod spec to JSON
	podSpecBytes, err := json.Marshal(podSpec)
	if err != nil {
		return "", fmt.Errorf("failed to serialize pod spec: %w", err)
	}

	// Compute the hash
	hash := sha256.Sum256(podSpecBytes)

	// Convert the hash to a hex string
	return hex.EncodeToString(hash[:]), nil
}
