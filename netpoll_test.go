//
//
//

package netpoll

import (
	"sync"
	"testing"
	"time"
)

func TestPollerStop(t *testing.T) {
	var wg sync.WaitGroup

	p, err := New(time.Second)
	if err != nil {
		t.Errorf("NEW: %v", err)
		return
	}
	wg.Add(1)
	go func() {
		p.Wait(1024)
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Second)
		p.Stop()
	}()

	p.Wait(1024)
	wg.Wait()
}
