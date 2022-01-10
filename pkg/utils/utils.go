package utils

import (
	"fmt"
	"os"
)

func GetUserHome() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error occured while getting user home from env: %s\n", err)
	}
	return userHome
}
