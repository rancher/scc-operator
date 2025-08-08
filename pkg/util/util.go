package util

import "os"

// primeSCCRegistrationHostURL tracks a global custom registration URL for online registrations
var primeSCCRegistrationHostURL = os.Getenv("PRIME_SCC_REGISTRATION_HOST_URL")

func HasGlobalPrimeRegistrationURL() bool {
	return primeSCCRegistrationHostURL != ""
}

func GetGlobalPrimeRegistrationURL() string {
	return primeSCCRegistrationHostURL
}
