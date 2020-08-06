//
// +build freebsd
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
	if self.poller, err = unix.Kqueue(); err != nil {
		return
	}
	self.event = 0
	event := []unix.Kevent_t{{Ident: uint64(self.event), Flags: unix.EV_ADD, Filter: unix.EVFILT_USER}}
	if _, err = unix.Kevent(self.poller, event, nil, nil); err != nil {
		unix.Close(self.poller)
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
	event := []unix.Kevent_t{{Ident: uint64(listen), Flags: unix.EV_ADD | unix.EV_CLEAR, Filter: unix.EVFILT_READ}}
	if _, err = unix.Kevent(self.poller, event, nil, nil); err != nil {
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
	event := []unix.Kevent_t{{Ident: uint64(fd), Flags: unix.EV_ADD | unix.EV_CLEAR, Filter: unix.EVFILT_READ}}
	if _, err = unix.Kevent(self.poller, event, nil, nil); err != nil {
		return
	}
	self.__set_fd_open(fd)
	self.added++
	return
}

func (self *Netpoll_t) DelFd(fd int) (err error) {
	self.mx.Lock()
	defer self.mx.Unlock()
	event := []unix.Kevent_t{{Ident: uint64(fd), Flags: unix.EV_DELETE}}
	if _, err = unix.Kevent(self.poller, event, nil, nil); err != nil {
		return
	}
	self.__set_fd_closed(fd)
	self.added--
	return
}

func (self *Netpoll_t) Wait(events_size int) (err error) {
	var conn int
	var count int
	events := make([]unix.Kevent_t, events_size)
	for {
		if count, err = unix.Kevent(self.poller, nil, events, nil); err != nil {
			if errno, _ := err.(unix.Errno); errno.Temporary() {
				continue
			}
			return
		}
		for i := 0; i < count; i++ {
			if int(events[i].Ident) == self.event && events[i].Filter == unix.EVFILT_USER {
				return
			}
			if int(events[i].Ident) == self.listen {
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
			// if events[i].Flags & unix.EV_EOF == unix.EV_EOF {
			// 	self.fd_event_close(int(events[i].Ident))
			// 	continue
			// }
			self.add_event(int(events[i].Ident))
		}
	}
}

func (self *Netpoll_t) Stop() (err error) {
	event := []unix.Kevent_t{{Ident: uint64(self.event), Filter: unix.EVFILT_USER, Fflags: unix.NOTE_TRIGGER}}
	_, err = unix.Kevent(self.poller, event, nil, nil)
	self.mx.Lock()
	self.running = false
	self.cond.Broadcast()
	self.mx.Unlock()
	return
}

func (self *Netpoll_t) Close() (err error) {
	unix.Close(self.listen)
	unix.Close(self.poller)
	return
}
