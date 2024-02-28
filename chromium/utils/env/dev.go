//go:build !production

package env

func IsDev() bool {
	return true
}

func Mode() string {
	return "dev"
}
