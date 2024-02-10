package chromium

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
)

// 创建文件服务
func RunFileServer(frontFiles embed.FS, frontPrefix string) string {
	// 创建文件服务
	http.Handle("/", http.FileServer(http.FS(&frontFileServerFs{
		prefix:     frontPrefix,
		frontFiles: frontFiles,
	})))

	// 检查可用端口
	defaultPort := 55556
	for {
		if CheckPortAvailability("127.0.0.1", defaultPort) {
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

func CheckPortAvailability(host string, port int) bool {
	ln, err := net.Listen("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

type frontFileServerFs struct {
	prefix     string
	frontFiles embed.FS
}

func (f *frontFileServerFs) Open(name string) (fs.File, error) {
	if name == "." {
		return f.frontFiles.Open(f.prefix)
	}
	if name == "/" {
		name = "index.html"
	}
	fmt.Println("[http server] open url ", name, " to file ", f.prefix+"/"+name)
	file, err := f.frontFiles.Open(f.prefix + "/" + name)
	if err != nil {
		fmt.Println("[http server] open url fail ", name, " error ", err)
	}
	return file, err
}
