package utils

import (
	"os"
	"strings"
)

func ExecutableName() string {
	executable, err := os.Executable()
	if err != nil {
		return "camel-pad"
	}

	parts := strings.Split(executable, string(os.PathSeparator))

	return parts[len(parts)-1]
}
