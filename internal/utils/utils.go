package utils

import (
	"os"
)

// Contains checks if given value is included in given slice.
func Contains(set []string, str string) bool {
	// a map may be faster
	for _, s := range set {
		if s == str {
			return true
		}
	}
	return false
}

// HomeDir returns platform-specific path to user's home directory.
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
