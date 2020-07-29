//
// +build linux
//

package netpoll

import (
	"net"
	"sync"
	"time"

	"github.com/ondi/go-cache"
	"golang.org/x/sys/unix"
)

func New(ttl time.Duration) (self *Netpoll_t, err error) {
	self = &Netpoll_t{
		listen: -1,
		ttl:    ttl,
		ready:  cache.New(),
	}
	self.cond = sync.NewCond(&self.mx)
	if self.poller, err = unix.EpollCreate1(0); err != nil {
		return
	}
	if self.event, err = unix.Eventfd(0, 0); err != nil {
		unix.Close(self.poller)
		return
	}
	if err = unix.EpollCtl(self.poller, unix.EPOLL_CTL_ADD, self.event,
		&unix.EpollEvent{Fd: int32(self.event), Events: unix.EPOLLIN},
	); err != nil {
		unix.Close(self.poller)
		unix.Close(self.event)
		return
	}
	self.running = true
	return
}

func (self *Netpoll_t) Listen(ip string, port int, zone uint32, backlog int) (err error) {
	var listen int
	if listen, err = unix.Socket(unix.AF_INET6, unix.SOCK_STREAM|unix.SOCK_NONBLOCK, 0); err != nil {
		return
	}
	addr := unix.SockaddrInet6{Port: port, ZoneId: zone}
	copy(addr.Addr[:], net.ParseIP(ip).To16())
	if err = unix.Bind(listen, &addr); err != nil {
		unix.Close(listen)
		return
	}
	if err = unix.Listen(listen, backlog); err != nil {
		unix.Close(listen)
		return
	}
	if err = unix.EpollCtl(self.poller, unix.EPOLL_CTL_ADD, listen,
		&unix.EpollEvent{Fd: int32(listen), Events: unix.EPOLLIN | unix.EPOLLRDHUP | unix.EPOLLET},
	); err != nil {
		unix.Close(listen)
		return
	}
	self.listen = listen
	return
}

func (self *Netpoll_t) AddFd(fd int) (err error) {
	self.mx.Lock()
	defer self.mx.Unlock()
	if err = unix.SetNonblock(fd, true); err != nil {
		return
	}
	if err = unix.EpollCtl(self.poller, unix.EPOLL_CTL_ADD, fd,
		&unix.EpollEvent{Fd: int32(fd), Events: unix.EPOLLIN | unix.EPOLLRDHUP | unix.EPOLLET},
	); err != nil {
		return
	}
	self.__set_fd_open(fd)
	self.added++
	return
}

func (self *Netpoll_t) DelFd(fd int) (err error) {
	self.mx.Lock()
	defer self.mx.Unlock()
	if err = unix.EpollCtl(self.poller, unix.EPOLL_CTL_DEL, fd, nil); err != nil {
		return
	}
	self.__set_fd_closed(fd)
	self.added--
	return
}

func (self *Netpoll_t) Wait(events_size int) (err error) {
	var conn int
	var count int
	events := make([]unix.EpollEvent, events_size)
	for {
		if count, err = unix.EpollWait(self.poller, events, -1); err != nil {
			if errno, _ := err.(unix.Errno); errno.Temporary() {
				continue
			}
			return
		}
		for i := 0; i < count; i++ {
			if int(events[i].Fd) == self.event {
				return
			}
			if int(events[i].Fd) == self.listen {
				for {
					if conn, _, err = unix.Accept(self.listen); err != nil {
						if errno, _ := err.(unix.Errno); errno.Temporary() {
							break
						}
						return
					}
					if self.AddFd(conn) != nil {
						unix.Close(conn)
					}
				}
				continue
			}
			if events[i].Events&unix.EPOLLHUP == unix.EPOLLHUP {
				self.set_fd_closed(int(events[i].Fd))
				continue
			}
			self.AddEvent(int(events[i].Fd))
		}
	}
}

func (self *Netpoll_t) Stop() (err error) {
	_, err = unix.Write(self.event, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	self.mx.Lock()
	self.running = false
	self.cond.Broadcast()
	self.mx.Unlock()
	return
}

func (self *Netpoll_t) Close() (err error) {
	unix.Close(self.listen)
	unix.Close(self.event)
	unix.Close(self.poller)
	return
}
