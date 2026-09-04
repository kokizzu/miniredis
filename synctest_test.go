//go:build go1.25

// go.mod says go 1.17, which defaults GODEBUG=asynctimerchan=1; synctest
// refuses to run with that. Drop this once go.mod is >= 1.23.
//go:debug asynctimerchan=0

package miniredis

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/alicebob/miniredis/v2/proto"
)

// Inside a testing/synctest bubble time.Now() is a fake clock that only
// advances when every goroutine in the bubble is blocked. Dial() keeps the
// server in the bubble (no TCP listener), and ClockTTL() makes expiry follow
// that fake clock, so a plain time.Sleep() expires keys.
func TestSynctestClockTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := NewMiniRedis()
		defer m.Close()
		m.ClockTTL(true)

		conn, err := m.Dial()
		ok(t, err)
		c := proto.NewClient(conn)
		defer c.Close()

		mustOK(t, c, "SET", "foo", "bar", "EX", "100")
		mustDo(t, c, "TTL", "foo", proto.Int(100))

		time.Sleep(30 * time.Second)
		mustDo(t, c, "TTL", "foo", proto.Int(70))
		mustDo(t, c, "GET", "foo", proto.String("bar"))

		time.Sleep(71 * time.Second)
		mustNil(t, c, "GET", "foo")
		mustDo(t, c, "EXISTS", "foo", proto.Int(0))
	})
}

// Blocking commands time out on the bubble's fake clock, so the test doesn't
// wait for real time.
func TestSynctestBlockingTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := NewMiniRedis()
		defer m.Close()

		conn, err := m.Dial()
		ok(t, err)
		c := proto.NewClient(conn)
		defer c.Close()

		start := time.Now()
		mustNilList(t, c, "BLPOP", "q", "10")
		equals(t, 10*time.Second, time.Since(start))
	})
}
