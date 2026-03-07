package tunnel

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var bwPattern = regexp.MustCompile(`^\s*(\d+)\s*([kmg]?)(bit|bps)?\s*$`)

type sessionStats struct {
	startedAt       time.Time
	upstreamBytes   atomic.Int64
	downstreamBytes atomic.Int64
}

func newSessionStats() *sessionStats {
	return &sessionStats{startedAt: time.Now()}
}

func (s *sessionStats) addUpstream(n int) {
	s.upstreamBytes.Add(int64(n))
}

func (s *sessionStats) addDownstream(n int) {
	s.downstreamBytes.Add(int64(n))
}

type rateEnforcer struct {
	bytesPerSecond int64
	startedAt      time.Time
	transferred    int64
	mu             sync.Mutex
}

func newRateEnforcer(bytesPerSecond int64) *rateEnforcer {
	return &rateEnforcer{
		bytesPerSecond: bytesPerSecond,
		startedAt:      time.Now(),
	}
}

func (r *rateEnforcer) accountAndThrottle(n int) {
	if r == nil || r.bytesPerSecond <= 0 || n <= 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.transferred += int64(n)
	expected := time.Duration(float64(r.transferred) / float64(r.bytesPerSecond) * float64(time.Second))
	elapsed := time.Since(r.startedAt)
	if expected > elapsed {
		time.Sleep(expected - elapsed)
	}
}

type clientSession struct {
	conn            net.Conn
	stats           *sessionStats
	upstreamLimiter *rateEnforcer
	downLimiter     *rateEnforcer
}

func newClientSession(conn net.Conn, bytesPerSecond int64) *clientSession {
	return &clientSession{
		conn:            conn,
		stats:           newSessionStats(),
		upstreamLimiter: newRateEnforcer(bytesPerSecond),
		downLimiter:     newRateEnforcer(bytesPerSecond),
	}
}

func parseBandwidthLimit(limit string) (int64, error) {
	limit = strings.TrimSpace(strings.ToLower(limit))
	if limit == "" || limit == "0" || limit == "0bit" || limit == "0bps" {
		return 0, nil
	}

	m := bwPattern.FindStringSubmatch(limit)
	if len(m) != 4 {
		return 0, fmt.Errorf("invalid bandwidth format %q", limit)
	}

	value, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid bandwidth value %q", m[1])
	}

	mult := int64(1)
	switch m[2] {
	case "k":
		mult = 1000
	case "m":
		mult = 1000 * 1000
	case "g":
		mult = 1000 * 1000 * 1000
	}

	bitsPerSec := value * mult
	if bitsPerSec == 0 {
		return 0, nil
	}
	return bitsPerSec / 8, nil
}
