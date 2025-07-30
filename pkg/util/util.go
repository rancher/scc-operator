package util

import "os"

// Define constants for clarity and reusability
const (
	KB  = 1024
	MiB = 1024 * KB // 1 MiB = 1,048,576 bytes
)

func BytesToMiBRounded(bytes int) int {
	// Handle zero or negative bytes gracefully to avoid issues with (bytes + MiB - 1)
	if bytes <= 0 {
		return 0
	}
	return (bytes + MiB - 1) / MiB
}

// primeSCCRegistrationHostURL tracks a global custom registration URL for online registrations
var primeSCCRegistrationHostURL = os.Getenv("PRIME_SCC_REGISTRATION_HOST_URL")

func HasGlobalPrimeRegistrationURL() bool {
	return primeSCCRegistrationHostURL != ""
}

func GetGlobalPrimeRegistrationURL() string {
	return primeSCCRegistrationHostURL
}
