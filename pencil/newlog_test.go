package pencil

import (
	"testing"
	"time"
)

func BenchmarkXLog(b *testing.B) {
	for i := 0; i < b.N; i++ {
		XLog()()
	}
}

func TestXLogMinute(t *testing.T) {
	XLogMinute(time.Minute / 3)()
}
