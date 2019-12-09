package utils

import "os"

func Contains(set []string, str string) bool {
	// a map may be faster
	for _, s := range set {
		if s == str {
			return true
		}
	}
	return false
}

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}
