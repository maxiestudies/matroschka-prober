package prober

import (
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

func (p *Prober) receiver() {
	defer p.udpConn.Close()

	recvBuffer := make([]byte, p.mtu)
	for {
		select {
		case <-p.stop:
			return
		default:
		}

		_, err := p.udpConn.Read(recvBuffer)
		now := time.Now().UnixNano()
		if err != nil {
			log.Errorf("Unable to read from UDP socket: %v", err)
			return
		}

		atomic.AddUint64(&p.probesReceived, 1)

		pkt, err := unmarshal(recvBuffer)
		if err != nil {
			log.Errorf("Unable to unmarshal message: %v", err)
			return
		}

		err = p.transitProbes.remove(pkt.SequenceNumber)
		if err != nil {
			// Probe was count as lost, so we ignore it from here on
			continue
		}

		rtt := now - pkt.TimeStamp
		if p.timedOut(rtt) {
			// Probe arrived late. rttTimoutChecker() will clean up after it. So we ignore it from here on
			atomic.AddUint64(&p.latePackets, 1)
			continue
		}

		p.measurements.AddRecv(pkt.TimeStamp, uint64(rtt), p.cfg.MeasurementLengthMS)
	}
}

func (p *Prober) timedOut(s int64) bool {
	return s > int64(msToNS(p.cfg.TimeoutMS))
}

func msToNS(s uint64) uint64 {
	return s * 1000000
}
