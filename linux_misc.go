//
// +build linux
//

package netpoll

import (
	"strconv"

	"golang.org/x/sys/unix"
)

func EVENTS(in uint32) (res []string) {
	for in != 0 {
		switch {
		case in&unix.EPOLLIN == unix.EPOLLIN:
			in ^= unix.EPOLLIN
			res = append(res, "EPOLLIN")
		case in&unix.EPOLLOUT == unix.EPOLLOUT:
			in ^= unix.EPOLLOUT
			res = append(res, "EPOLLOUT")
		case in&unix.EPOLLHUP == unix.EPOLLHUP:
			in ^= unix.EPOLLHUP
			res = append(res, "EPOLLHUP")
		case in&unix.EPOLLERR == unix.EPOLLERR:
			in ^= unix.EPOLLERR
			res = append(res, "EPOLLERR")
		case in&unix.EPOLLRDHUP == unix.EPOLLRDHUP:
			in ^= unix.EPOLLRDHUP
			res = append(res, "EPOLLRDHUP")
		default:
			res = append(res, "UNKNOWN("+strconv.Itoa(int(in))+")")
			return
		}
	}
	return
}
