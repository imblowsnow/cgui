package utils

import "os"

func IsDev() bool {
	return os.Getenv("mode") == "dev"
}

func Mode() string {
	return os.Getenv("mode")
}
