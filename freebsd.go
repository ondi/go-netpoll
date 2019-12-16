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
	// if self.event, err = unix.Eventfd(0, 0); err != nil {
	// 	unix.Close(self.poller)
	// 	return
	// }
	// var event [1]unix.Kevent_t
	// unix.SetKevent(&event[0], self.event, unix.EV_ADD, unix.EVFILT_READ)
	// if _, err = unix.Kevent(self.poller, event[:], nil, nil); err != nil {
	// 	unix.Close(self.poller)
	// 	unix.Close(self.event)
	// 	return
	// }
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
	unix.SetKevent(&event[0], listen, unix.EV_ADD, unix.EVFILT_READ)
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
	unix.SetKevent(&event[0], fd, unix.EV_ADD, unix.EVFILT_READ)
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
	unix.SetKevent(&event[0], fd, unix.EV_DELETE, 0)
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
			if int(events[i].Ident) == self.event {
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
			if events[i].Filter & unix.EVFILT_VNODE == unix.EVFILT_VNODE /*&& events[i].Ffilter & unix.NOTE_CLOSE == unix.NOTE_CLOSE*/ {
				self.fd_event_close(int(events[i].Ident))
				continue
			}
			self.AddEvent(int(events[i].Ident))
		}
	}
}

func (self * Netpoll_t) Stop() (err error) {
	_, err = unix.Write(self.event, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	return
}

func (self * Netpoll_t) Close() (err error) {
	unix.Close(self.listen)
	unix.Close(self.event)
	unix.Close(self.poller)
	return
}
