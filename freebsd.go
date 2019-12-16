//
// +build freebsd
//

package netpoll

import "net"
import "time"
import "sync"

import "golang.org/x/sys/unix"
import "github.com/ondi/go-cache"

func New(closed_ttl time.Duration) (self * Netpoll_t, err error) {
	self = &Netpoll_t{
		listen: -1,
		closed_ttl: closed_ttl,
		ready: cache.New(),
	}
	self.cond = sync.NewCond(&self.mx)
	if self.poller, err = unix.Kqueue(); err != nil {
		return
	}
	self.event = 0
	var event [1]unix.Kevent_t
	unix.SetKevent(&event[0], self.event, unix.EVFILT_USER, unix.EV_ADD)
	event[0].Fflags = unix.NOTE_FFNOP
	if _, err = unix.Kevent(self.poller, event[:], nil, nil); err != nil {
		unix.Close(self.poller)
		return
	}
	return
}

func (self * Netpoll_t) Listen(ip string, port int, zone uint32, backlog int) (err error) {
	var listen int
	if listen, err = unix.Socket(unix.AF_INET6, unix.SOCK_STREAM | unix.SOCK_NONBLOCK, 0); err != nil {
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
	var event [1]unix.Kevent_t
	unix.SetKevent(&event[0], listen, unix.EVFILT_READ, unix.EV_ADD)
	if _, err = unix.Kevent(self.poller, event[:], nil, nil); err != nil {
		unix.Close(listen)
		return
	}
	self.listen = listen
	return
}

func (self * Netpoll_t) Add(fd int) (err error) {
	self.mx.Lock()
	defer self.mx.Unlock()
	if err = unix.SetNonblock(fd, true); err != nil {
		return
	}
	var event [1]unix.Kevent_t
	unix.SetKevent(&event[0], fd, unix.EVFILT_READ, unix.EV_ADD)
	if _, err = unix.Kevent(self.poller, event[:], nil, nil); err != nil {
		return
	}
	self.__fd_event_open(fd)
	self.added++
	return
}

func (self * Netpoll_t) Del(fd int) (err error) {
	self.mx.Lock()
	defer self.mx.Unlock()
	var event [1]unix.Kevent_t
	unix.SetKevent(&event[0], fd, 0, unix.EV_DELETE)
	if _, err = unix.Kevent(self.poller, event[:], nil, nil); err != nil {
		return
	}
	self.__fd_event_close(fd)
	self.added--
	return
}

func (self * Netpoll_t) Wait(events_size int) (err error) {
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
					if self.Add(conn) != nil {
						unix.Close(conn)
					}
				}
				continue
			}
			if events[i].Filter & unix.EVFILT_VNODE == unix.EVFILT_VNODE && events[i].Fflags & unix.NOTE_CLOSE_WRITE == unix.NOTE_CLOSE_WRITE {
				self.fd_event_close(int(events[i].Ident))
				continue
			}
			self.AddEvent(int(events[i].Ident))
		}
	}
}

func (self * Netpoll_t) Stop() (err error) {
	var event [1]unix.Kevent_t
	unix.SetKevent(&event[0], self.event, unix.EVFILT_USER, 0)
	event[0].Fflags = unix.NOTE_TRIGGER
	_, err = unix.Kevent(self.poller, event[:], nil, nil)
	return
}

func (self * Netpoll_t) Close() (err error) {
	unix.Close(self.listen)
	unix.Close(self.poller)
	return
}
