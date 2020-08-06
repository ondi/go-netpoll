//
// Repeat Read() and Write() until EAGAIN.
// Local cache prevents fd to run on different threads,
// provides a read queue between epoll_wait() and Read()
// with short lived state.Data for slow clients which may interact like this:
// WS_FRAME_PART_1 -> EAGAIN -> WS_FRAME_PART_2 -> EAGAIN -> ... -> WS_FRAME_PART_N
//
// USAGE:
// if poller, err = netpoll.New(ttl); err != nil {
// 	return
// }
// for i := 0; i < publishers; i++ {
// 	go poller.Wait(poll_size)
// }
// for i := 0; i < consumers; i++ {
// 	go poller.Read(server.ws_read)
// }
// ...
// net_conn should be stored somewhere with fd
// if fd, err = netpoll.GetFd(net_conn); err != nil {
// 	return
// }
// if err = poller.AddFd(fd); err != nil {
// 	 return
// }
//
// see 'Possible Pitfalls and Ways to Avoid Them' for details
// https://linux.die.net/man/4/epoll
//

package netpoll

import (
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/ondi/go-cache"
)

const (
	FLAG_RUNNING uint64 = 1 << 63
	FLAG_CLOSED  uint64 = 1 << 62
)

type READ func(int, *State_t)

type State_t struct {
	updated time.Time
	events  uint64
	Data    interface{}
}

type Netpoll_t struct {
	poller int
	event  int
	listen int
	ttl    time.Duration

	mx      sync.Mutex
	cond    *sync.Cond
	ready   *cache.Cache_t
	added   int
	running bool
}

func (self *Netpoll_t) __set_fd_open(fd int) {
	self.ready.UpdateBack(fd, func(value interface{}) interface{} {
		value.(*State_t).events &= ^FLAG_CLOSED
		return value
	})
}

func (self *Netpoll_t) __set_fd_closed(fd int) {
	it, ok := self.ready.PushBack(fd, func() interface{} {
		return &State_t{updated: time.Now(), events: FLAG_CLOSED}
	})
	if !ok {
		it.Value().(*State_t).updated = time.Now()
		it.Value().(*State_t).events = FLAG_CLOSED
	}
}

func (self *Netpoll_t) set_fd_closed(fd int) {
	self.mx.Lock()
	self.__set_fd_closed(fd)
	self.mx.Unlock()
}

func (self *Netpoll_t) add_event(fd int) {
	self.mx.Lock()
	it, ok := self.ready.PushBack(fd, func() interface{} { return &State_t{updated: time.Now(), events: 1} })
	if ok {
		self.cond.Signal()
		self.mx.Unlock()
		return
	}
	if it.Value().(*State_t).events&FLAG_CLOSED == FLAG_CLOSED {
		self.mx.Unlock()
		return
	}
	it.Value().(*State_t).updated = time.Now()
	it.Value().(*State_t).events++
	if it.Value().(*State_t).events&FLAG_RUNNING == 0 {
		self.cond.Signal()
	}
	self.mx.Unlock()
}

func (self *Netpoll_t) Read(fn READ) {
	self.mx.Lock()
	for self.running {
		self.cond.Wait()
	loop:
		now := time.Now()
		for i := 0; i < self.ready.Size(); {
			it := self.ready.Front()
			state := it.Value().(*State_t)
			if state.events&FLAG_RUNNING == FLAG_RUNNING {
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			if state.events & ^FLAG_CLOSED == 0 {
				if now.Sub(state.updated) > self.ttl {
					self.ready.Remove(it.Key())
					continue
				}
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			state.events &= FLAG_CLOSED
			state.events |= FLAG_RUNNING
			cache.MoveBefore(it, self.ready.End())
			self.mx.Unlock()
			fn(it.Key().(int), it.Value().(*State_t))
			self.mx.Lock()
			state.events &= ^FLAG_RUNNING
			goto loop
		}
	}
	self.mx.Unlock()
}

func (self *Netpoll_t) SizeAdded() (res int) {
	self.mx.Lock()
	res = self.added
	self.mx.Unlock()
	return
}

func (self *Netpoll_t) SizeReady() (res int) {
	self.mx.Lock()
	res = self.ready.Size()
	self.mx.Unlock()
	return
}

func GetFd(conn net.Conn) (fd int, err error) {
	if tcp_conn, ok := conn.(*net.TCPConn); ok {
		var raw syscall.RawConn
		if raw, err = tcp_conn.SyscallConn(); err != nil {
			return
		}
		raw.Control(func(c uintptr) { fd = int(c) })
		return
	}
	return -1, fmt.Errorf("not a *net.TCPConn")
}
