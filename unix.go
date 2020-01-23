//
// +build linux freebsd
//

package netpoll

import "golang.org/x/sys/unix"

import "net"

func GetIpPort(sa unix.Sockaddr) (ip string, port int) {
	if ip4, ok := sa.(* unix.SockaddrInet4); ok {
		ip = net.IP(ip4.Addr[:]).String()
		port = ip4.Port
		return
	}
	if ip6, ok := sa.(* unix.SockaddrInet6); ok {
		ip = net.IP(ip6.Addr[:]).String()
		port = ip6.Port
	}
	return
}

func Getpeername(fd int) (ip string, port int, err error) {
	var sa unix.Sockaddr
	if sa, err = unix.Getpeername(fd); err != nil {
		return
	}
	ip, port = GetIpPort(sa)
	return
}

func Getsockname(fd int) (ip string, port int, err error) {
	var sa unix.Sockaddr
	if sa, err = unix.Getsockname(fd); err != nil {
		return
	}
	ip, port = GetIpPort(sa)
	return
}

type FD int

func (self FD) Read(p []byte) (n int, err error) {
	if n, err = unix.Read(int(self), p); err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, unix.EOWNERDEAD
	}
	return
}

func (self FD) Write(p []byte) (int, error) {
	return unix.Write(int(self), p)
}
