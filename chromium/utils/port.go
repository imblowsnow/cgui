package utils

import (
	"net"
	"strconv"
)

func CheckPortAvailability(host string, port int) bool {
	ln, err := net.Listen("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
