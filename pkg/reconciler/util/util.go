package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// MergeEnvVars merges one slice of corev1.EnvVar into another slice of corev1.EnvVar
// if overwriteValues is false, this function will return an error if a duplicate EnvVar name is encountered
// if overwriteValues is true, this function will overwrite the existing value with the new value if a duplicate is encountered
func MergeEnvVars(new []corev1.EnvVar, into []corev1.EnvVar, overwriteValues bool) ([]corev1.EnvVar, error) {
	originalEnvs := make(map[string]int)

	for i, o := range into {
		originalEnvs[o.Name] = i
	}

	for _, n := range new {
		_, exists := originalEnvs[n.Name]

		switch {
		case exists && overwriteValues:
			into[originalEnvs[n.Name]] = n
		case exists && !overwriteValues:
			return []corev1.EnvVar{}, fmt.Errorf("environment variable %q already exists", n.Name)
		default:
			into = append(into, n)
		}
	}

	return into, nil
}
