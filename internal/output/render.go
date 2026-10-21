package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/marcelsecco1/dnsquerypulse/internal/dns"
	"github.com/marcelsecco1/dnsquerypulse/internal/stats"
)

func Results(w io.Writer, rs []dns.Result, format string) error {
	switch format {
	case "json":
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(rs)
	case "csv":
		c := csv.NewWriter(w)
		_ = c.Write([]string{"domain", "query_type", "resolver", "transport", "started_at", "duration_ms", "status", "rcode", "answers", "error_kind", "error_message"})
		for _, r := range rs {
			_ = c.Write([]string{r.Domain, r.QueryType, r.Resolver, r.Transport, r.StartedAt.Format(time.RFC3339Nano), fmt.Sprintf("%.3f", float64(r.Duration)/float64(time.Millisecond)), r.Status, r.RCode, strings.Join(r.Answers, ";"), r.ErrorKind, r.ErrorMessage})
		}
		c.Flush()
		return c.Error()
	default:
		widths := []int{20, 8, 18, 12, 10, 12, 30}
		border := tableBorder(widths)
		fmt.Fprintln(w, border)
		writeTableRow(w, widths, []string{"DOMAIN", "TYPE", "RESOLVER", "LATENCY", "RCODE", "STATUS", "ANSWER"}, []string{"38;2;96;165;250", "38;2;96;165;250", "38;2;96;165;250", "38;2;96;165;250", "38;2;96;165;250", "38;2;96;165;250", "38;2;96;165;250"})
		fmt.Fprintln(w, border)
		for _, r := range rs {
			answers := r.Answers
			if len(answers) == 0 {
				answers = []string{"-"}
			}
			statusColor := statusColor(r.Status)
			rcodeColor := rcodeColor(r.RCode)
			for i, answer := range answers {
				values := []string{clip(r.Domain, 20), clip(r.QueryType, 8), clip(r.Resolver, 18), clip(durationText(r.Duration.Round(time.Microsecond)), 12), clip(r.RCode, 10), clip(r.Status, 12), clip(answer, 30)}
				colors := []string{"", "", "38;2;192;132;252", "38;2;34;211;238", rcodeColor, statusColor, statusColor}
				if i > 0 {
					for c := 0; c < 6; c++ {
						values[c] = ""
						colors[c] = ""
					}
				}
				writeTableRow(w, widths, values, colors)
			}
			fmt.Fprintln(w, border)
		}
		return nil
	}
}
func tableBorder(widths []int) string {
	var b strings.Builder
	b.WriteByte('+')
	for _, width := range widths {
		b.WriteString(strings.Repeat("-", width+2))
		b.WriteByte('+')
	}
	return b.String()
}
func writeTableRow(w io.Writer, widths []int, values, colors []string) {
	cells := make([]string, len(widths))
	for i, width := range widths {
		value := clip(values[i], width)
		padding := strings.Repeat(" ", max(0, width-len([]rune(value))))
		if colors[i] != "" && colorEnabled() {
			value = "\x1b[" + colors[i] + "m" + value + "\x1b[0m"
		}
		cells[i] = " " + value + padding + " "
	}
	fmt.Fprintf(w, "|%s|\n", strings.Join(cells, "|"))
}
func statusColor(status string) string {
	switch status {
	case "success":
		return "38;2;74;222;128"
	case "timeout":
		return "38;2;250;204;21"
	case "rcode":
		return "38;2;251;146;60"
	default:
		return "38;2;248;113;113"
	}
}
func rcodeColor(rcode string) string {
	switch rcode {
	case "NOERROR":
		return "38;2;74;222;128"
	case "NXDOMAIN", "FORMERR":
		return "38;2;250;204;21"
	case "SERVFAIL", "REFUSED":
		return "38;2;248;113;113"
	default:
		return ""
	}
}
func colorEnabled() bool { return os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" }

func Summary(w io.Writer, s stats.Summary) {
	const border = "+--------------------------------------------------------------------------+"
	fmt.Fprintln(w, border)
	fmt.Fprintf(w, "| %-72s |\n", "DNSQUERYPULSE SUMMARY")
	fmt.Fprintln(w, border)
	fmt.Fprintf(w, "| %-72s |\n", clip(fmt.Sprintf("Queries: %d  |  Success: %d  |  Failed: %d  |  Availability: %.1f%%", s.Total, s.Success, s.Errors+s.Timeouts, s.SuccessRate*100), 72))
	fmt.Fprintf(w, "| %-72s |\n", clip(fmt.Sprintf("Latency: p50 %s  |  p95 %s  |  p99 %s", durationText(s.P50.Round(time.Microsecond)), durationText(s.P95.Round(time.Microsecond)), durationText(s.P99.Round(time.Microsecond))), 72))
	fmt.Fprintf(w, "| %-72s |\n", clip(fmt.Sprintf("Consecutive failures: %d  |  RCODEs: %s", s.ConsecutiveFailures, formatRCodes(s.RCodes)), 72))
	fmt.Fprintln(w, border)
}
func formatRCodes(rcodes map[string]int) string {
	if len(rcodes) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(rcodes))
	for code := range rcodes {
		keys = append(keys, code)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, code := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", code, rcodes[code]))
	}
	return strings.Join(parts, ", ")
}
func Report(w io.Writer, rs []dns.Result, s stats.Summary, format string) error {
	if format == "json" {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(struct {
			Results []dns.Result  `json:"results"`
			Summary stats.Summary `json:"summary"`
		}{rs, s})
	}
	if err := Results(w, rs, format); err != nil {
		return err
	}
	if format != "csv" {
		Summary(w, s)
	}
	return nil
}
func Banner(w io.Writer) {
	const width = 72
	border := "+" + strings.Repeat("-", width) + "+"
	fmt.Fprintln(w, border)
	fmt.Fprintf(w, "|%s|\n", strings.Repeat(" ", width))
	name := "DNSQUERYPULSE"
	left := (width - len(name)) / 2
	wordmark := strings.Repeat(" ", left) + gradient(name) + strings.Repeat(" ", width-left-len(name))
	fmt.Fprintf(w, "|%s|\n", wordmark)
	subtitle := "DNS diagnostics  |  monitoring  |  benchmarking"
	subLeft := (width - len(subtitle)) / 2
	fmt.Fprintf(w, "|%s%s%s|\n", strings.Repeat(" ", subLeft), subtitle, strings.Repeat(" ", width-subLeft-len(subtitle)))
	fmt.Fprintf(w, "|%s|\n", strings.Repeat(" ", width))
	fmt.Fprintln(w, border)
}
func gradient(s string) string {
	if !colorEnabled() {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		p := float64(i) / float64(max(1, len(runes)-1))
		// Sweep through deep blue, vivid blue, and cyan-blue.
		var red, green, blue int
		if p < 0.5 {
			t := p * 2
			red, green, blue = int(20+10*t), int(55+85*t), int(210+35*t)
		} else {
			t := (p - 0.5) * 2
			red, green, blue = int(30-20*t), int(140+65*t), 245
		}
		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm%c", red, green, blue, r)
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

func clip(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	if limit <= 3 {
		return string(r[:limit])
	}
	return string(r[:limit-3]) + "..."
}

func durationText(d time.Duration) string {
	return strings.ReplaceAll(d.String(), "µs", "us")
}
