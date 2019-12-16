//
//
//

package netpoll

import "time"
import "testing"

func TestEpoll1(t * testing.T) {
	p, err := New(time.Second)
	if err != nil {
		t.Errorf("NEW: %v", err)
		return
	}
	
	go func() {
		p.Wait(1024)
		t.Log("LOOP END")
	}()
	
	go func() {
		time.Sleep(time.Second)
		p.Stop()
	}()
	
	p.Wait(1024)
}
