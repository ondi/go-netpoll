//
// +build freebsd
//

package netpoll

import "strconv"

import "golang.org/x/sys/unix"

func FILTER(in int16) (res []string) {
	for in != 0 {
		switch {
		case in & unix.EVFILT_READ == unix.EVFILT_READ:
			in ^= unix.EVFILT_READ
			res = append(res, "EVFILT_READ")
		case in & unix.EVFILT_WRITE == unix.EVFILT_WRITE:
			in ^= unix.EVFILT_WRITE
			res = append(res, "EVFILT_WRITE")
		case in & unix.EVFILT_VNODE == unix.EVFILT_VNODE:
			in ^= unix.EVFILT_VNODE
			res = append(res, "EVFILT_VNODE")
		case in & unix.EVFILT_USER == unix.EVFILT_USER:
			in ^= unix.EVFILT_USER
			res = append(res, "EVFILT_USER")
		default:
			res = append(res, "UNKNOWN(" + strconv.Itoa(int(in)) + ")")
			return
		}
	}
	return
}

func FFLAGS(in uint32) (res []string) {
	for in != 0 {
		switch {
		case in & unix.NOTE_CLOSE == unix.NOTE_CLOSE:
			in ^= unix.NOTE_CLOSE
			res = append(res, "NOTE_CLOSE")
		case in & unix.NOTE_CLOSE_WRITE == unix.NOTE_CLOSE_WRITE:
			in ^= unix.NOTE_CLOSE_WRITE
			res = append(res, "NOTE_CLOSE_WRITE")
		default:
			res = append(res, "UNKNOWN(" + strconv.Itoa(int(in)) + ")")
			return
		}
	}
	return
}

func FLAGS(in uint16) (res []string) {
	for in != 0 {
		switch {
		case in & unix.EV_ADD == unix.EV_ADD:
			in ^= unix.EV_ADD
			res = append(res, "EV_ADD")
		case in & unix.EV_ENABLE == unix.EV_ENABLE:
			in ^= unix.EV_ENABLE
			res = append(res, "EV_ENABLE")
		case in & unix.EV_DISABLE == unix.EV_DISABLE:
			in ^= unix.EV_DISABLE
			res = append(res, "EV_DISABLE")
		case in & unix.EV_DISPATCH == unix.EV_DISPATCH:
			in ^= unix.EV_DISPATCH
			res = append(res, "EV_DISPATCH")
		case in & unix.EV_DELETE == unix.EV_DELETE:
			in ^= unix.EV_DELETE
			res = append(res, "EV_DELETE")
		case in & unix.EV_RECEIPT == unix.EV_RECEIPT:
			in ^= unix.EV_RECEIPT
			res = append(res, "EV_RECEIPT")
		case in & unix.EV_ONESHOT == unix.EV_ONESHOT:
			in ^= unix.EV_ONESHOT
			res = append(res, "EV_ONESHOT")
		case in & unix.EV_CLEAR == unix.EV_CLEAR:
			in ^= unix.EV_CLEAR
			res = append(res, "EV_CLEAR")
		case in & unix.EV_EOF == unix.EV_EOF:
			in ^= unix.EV_EOF
			res = append(res, "EV_EOF")
		case in & unix.EV_ERROR == unix.EV_ERROR:
			in ^= unix.EV_ERROR
			res = append(res, "EV_ERROR")
		default:
			res = append(res, "UNKNOWN(" + strconv.Itoa(int(in)) + ")")
			return
		}
	}
	return
}
