package stats

import (
	"github.com/marcelsecco1/dnsquerypulse/internal/dns"
	"math"
	"sort"
	"time"
)

type Summary struct {
	Total               int            `json:"total"`
	Success             int            `json:"success"`
	Timeouts            int            `json:"timeouts"`
	Errors              int            `json:"errors"`
	RCodes              map[string]int `json:"rcodes"`
	SuccessRate         float64        `json:"success_rate"`
	Mean                time.Duration  `json:"mean_ns"`
	P50                 time.Duration  `json:"p50_ns"`
	P95                 time.Duration  `json:"p95_ns"`
	P99                 time.Duration  `json:"p99_ns"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
}

func Summarize(rs []dns.Result) Summary {
	s := Summary{Total: len(rs), RCodes: map[string]int{}}
	var ds []time.Duration
	for _, r := range rs {
		if r.RCode != "-" {
			s.RCodes[r.RCode]++
		}
		if r.Status == "success" {
			s.Success++
			ds = append(ds, r.Duration)
		} else if r.Status == "timeout" {
			s.Timeouts++
		} else {
			s.Errors++
		}
	}
	if s.Total > 0 {
		s.SuccessRate = float64(s.Success) / float64(s.Total)
	}
	if s.Success == 0 {
		return s
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	var n time.Duration
	for _, d := range ds {
		n += d
	}
	s.Mean = n / time.Duration(len(ds))
	s.P50 = percentile(ds, .50)
	s.P95 = percentile(ds, .95)
	s.P99 = percentile(ds, .99)
	for i := len(rs) - 1; i >= 0; i-- {
		if rs[i].Status == "success" {
			break
		}
		s.ConsecutiveFailures++
	}
	return s
}
func percentile(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(ds)))) - 1
	if idx < 0 {
		idx = 0
	}
	return ds[idx]
}
