package front

import (
	"log"
	"main/chromium/utils"
	"net/http"
	"strconv"
)

func RunLocalFileServer(path string) string {
	// 创建文件服务
	http.Handle("/", http.FileServer(http.Dir(path)))

	// 检查可用端口
	defaultPort := startPort
	for {
		if utils.CheckPortAvailability("127.0.0.1", defaultPort) {
			break
		}
		defaultPort++
	}

	var addr = "127.0.0.1:" + strconv.Itoa(defaultPort)

	// 启动服务
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	return addr
}
