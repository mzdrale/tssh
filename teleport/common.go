package teleport

import (
	"os/user"
)

// GetLocalUserHomeDir - get user's home dir
func GetLocalUserHomeDir() (string, error) {
	user, err := user.Current()

	if err != nil {
		return "", err
	}

	return user.HomeDir, nil
}
