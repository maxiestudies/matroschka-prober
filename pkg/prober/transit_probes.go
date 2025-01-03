package prober

import (
	"fmt"
	"sync"
)

type transitProbes struct {
	m map[uint64]int64 // index is the unix time in seconds, value is the sequence number
	l sync.RWMutex
}

func (t *transitProbes) add(p *probe) {
	t.l.Lock()
	defer t.l.Unlock()
	t.m[p.SequenceNumber] = p.TimeStamp
}

func (t *transitProbes) remove(s uint64) error {
	t.l.Lock()

	if _, ok := t.m[s]; !ok {
		t.l.Unlock()
		return fmt.Errorf("Sequence number %d not found", s)
	}

	delete(t.m, s)
	t.l.Unlock()
	return nil
}

func (t *transitProbes) getLt(lt int64) map[uint64]int64 {
	ret := make(map[uint64]int64)
	t.l.RLock()
	defer t.l.RUnlock()

	for s, ts := range t.m {
		if ts < lt {
			ret[s] = ts
		}
	}

	return ret
}

func newTransitProbes() *transitProbes {
	return &transitProbes{
		m: make(map[uint64]int64),
	}
}
