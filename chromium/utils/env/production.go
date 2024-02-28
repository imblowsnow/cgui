//go:build production

package env

func IsDev() bool {
	return false
}

func Mode() string {
	return "production"
}
