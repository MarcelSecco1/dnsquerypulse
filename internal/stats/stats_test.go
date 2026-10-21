package stats

import (
	"github.com/marcelsecco1/dnsquerypulse/internal/dns"
	"testing"
	"time"
)

func TestSummarizeUsesSuccessfulSamplesForLatency(t *testing.T) {
	rs := []dns.Result{{Status: "success", RCode: "NOERROR", Duration: time.Millisecond}, {Status: "timeout", RCode: "-"}, {Status: "success", RCode: "NOERROR", Duration: 3 * time.Millisecond}}
	s := Summarize(rs)
	if s.Total != 3 || s.Success != 2 || s.Timeouts != 1 || s.P50 != time.Millisecond || s.P99 != 3*time.Millisecond || s.SuccessRate != 2.0/3.0 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}
func TestConsecutiveFailuresCountFromLatestResult(t *testing.T) {
	s := Summarize([]dns.Result{{Status: "timeout"}, {Status: "success"}, {Status: "error"}})
	if s.ConsecutiveFailures != 1 {
		t.Fatalf("got %d consecutive failures", s.ConsecutiveFailures)
	}
}
