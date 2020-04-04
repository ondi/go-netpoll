//
// Repeat Read() and Write() untill EAGAIN fired,
// for streams use AddEvent() before breaking read cycle to be called again.
// Add() and Del() should share the same mutex with user fd cache.
// local cache prevents fd to run on different threads
// and provides a read queue between epoll_wait() and Read().
//
// RACES:
// fd may be deleted, closed and reopened right after epoll_wait()
// and before processing. Read() may get fd with outdated events.
// the following race condition present in code below:
// one thread may call Del(fd), Add(fd) while events from epoll_wait()
// still processing and closed flag already unset.
//
// see 'Possible Pitfalls and Ways to Avoid Them' for details
// https://linux.die.net/man/4/epoll
//

package netpoll

import (
	"net"
	"sync"
	"time"

	"github.com/ondi/go-cache"
)

const (
	FLAG_RUNNING uint64 = 1 << 63
)

type READ func(int)

type state_t struct {
	closed time.Time
	events uint64
}

type Netpoll_t struct {
	poller     int
	event      int
	listen     int
	closed_ttl time.Duration

	mx      sync.Mutex
	cond    *sync.Cond
	ready   *cache.Cache_t
	added   int
	running bool
}

func (self *Netpoll_t) __set_fd_open(fd int) {
	self.ready.UpdateBack(fd, func(value interface{}) interface{} {
		value.(*state_t).closed = time.Time{}
		return value
	})
}

func (self *Netpoll_t) __set_fd_closed(fd int) {
	if it, ok := self.ready.PushBack(fd, func() interface{} { return &state_t{closed: time.Now()} }); !ok {
		it.Value().(*state_t).closed = time.Now()
	}
}

func (self *Netpoll_t) set_fd_closed(fd int) {
	self.mx.Lock()
	self.__set_fd_closed(fd)
	self.mx.Unlock()
}

func (self *Netpoll_t) AddEvent(fd int) {
	self.mx.Lock()
	defer self.mx.Unlock()
	it, ok := self.ready.PushBack(fd, func() interface{} { return &state_t{events: 1} })
	if ok {
		self.cond.Signal()
		return
	}
	if it.Value().(*state_t).closed.IsZero() == false {
		return
	}
	it.Value().(*state_t).events++
	if it.Value().(*state_t).events&FLAG_RUNNING == 0 {
		self.cond.Signal()
	}
}

func (self *Netpoll_t) Read(fn READ) {
	self.mx.Lock()
	for self.running {
		self.cond.Wait()
	loop:
		now := time.Now()
		for i := 0; i < self.ready.Size(); {
			it := self.ready.Front()
			state := it.Value().(*state_t)
			if state.events&FLAG_RUNNING == FLAG_RUNNING {
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			if state.closed.IsZero() == false {
				if now.Sub(state.closed) > self.closed_ttl {
					self.ready.Remove(it.Key())
					continue
				}
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			if state.events == 0 {
				self.ready.Remove(it.Key())
				continue
			}
			state.events = FLAG_RUNNING
			cache.MoveBefore(it, self.ready.End())
			self.mx.Unlock()
			fn(it.Key().(int))
			self.mx.Lock()
			state.events &= ^FLAG_RUNNING
			goto loop
		}
	}
}

func (self *Netpoll_t) SizeAdded() int {
	self.mx.Lock()
	defer self.mx.Unlock()
	return self.added
}

func (self *Netpoll_t) SizeReady() int {
	self.mx.Lock()
	defer self.mx.Unlock()
	return self.ready.Size()
}

func GetFd(conn net.Conn) (fd int) {
	if tcp_conn, ok := conn.(*net.TCPConn); ok {
		if raw, err := tcp_conn.SyscallConn(); err == nil {
			raw.Control(
				func(c uintptr) {
					fd = int(c)
				},
			)
			return
		}
	}
	return -1
}
