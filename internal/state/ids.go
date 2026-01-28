package state

import (
	"crypto/sha256"
	"fmt"
)

// GenerateTargetID creates a unique ID for a target
func GenerateTargetID(targetType TargetType, name, path string) string {
	data := fmt.Sprintf("%s:%s:%s", targetType, name, path)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes (32 hex chars)
}

// GenerateServiceID creates a unique ID for a service
func GenerateServiceID(targetID, serviceName string) string {
	data := fmt.Sprintf("%s:%s", targetID, serviceName)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes (32 hex chars)
}
