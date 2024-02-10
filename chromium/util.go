package chromium

import "os"

// 获取当前目录
func GetCurrentDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return dir, nil
}

func GetCurrentBrowserFlagDir(name string) (string, error) {
	dir, err := GetCurrentDir()
	if err != nil {
		return "", err
	}
	return dir + "/.chrome/" + name, nil
}
