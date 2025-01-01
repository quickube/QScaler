package controller

import (
	"reflect"
	"testing"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
)

// TestGeneratePodTemplateHash tests the GeneratePodTemplateHash function
func TestGeneratePodTemplateHash(t *testing.T) {
	// Define a sample PodSpec for testing
	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "test-container",
				Image: "test-image",
				Env: []corev1.EnvVar{
					{Name: "TEST_ENV", Value: "value"},
				},
			},
		},
		RestartPolicy: corev1.RestartPolicyAlways,
	}

	// Expected hash calculation
	podSpecBytes, err := json.Marshal(podSpec)
	if err != nil {
		t.Fatalf("Failed to serialize PodSpec: %v", err)
	}
	expectedHash := sha256.Sum256(podSpecBytes)
	expectedHashString := hex.EncodeToString(expectedHash[:])

	// Call the function under test
	actualHash, err := GeneratePodTemplateHash(podSpec)
	if err != nil {
		t.Fatalf("GeneratePodTemplateHash returned an error: %v", err)
	}

	// Verify the result
	if !reflect.DeepEqual(expectedHashString, actualHash) {
		t.Errorf("Expected hash %s, got %s", expectedHashString, actualHash)
	}
}

func TestGeneratePodTemplateHash_Deterministic(t *testing.T) {
	// Define the pod spec to test
	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:latest",
			},
			{
				Name:  "sidecar",
				Image: "busybox:latest",
			},
		},
	}

	// Run GeneratePodTemplateHash twice with the same input
	hash1, err1 := GeneratePodTemplateHash(podSpec)
	hash2, err2 := GeneratePodTemplateHash(podSpec)

	// Assert no errors
	if err1 != nil {
		t.Fatalf("First call to GeneratePodTemplateHash() returned error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second call to GeneratePodTemplateHash() returned error: %v", err2)
	}

	// Assert the hashes are equal
	if hash1 != hash2 {
		t.Errorf("GeneratePodTemplateHash() returned different results: hash1 = %v, hash2 = %v", hash1, hash2)
	} else {
		t.Logf("Hashes match: %v", hash1)
	}
}
