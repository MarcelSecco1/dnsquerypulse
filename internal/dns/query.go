package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Domain       string        `json:"domain"`
	QueryType    string        `json:"query_type"`
	Resolver     string        `json:"resolver"`
	Transport    string        `json:"transport"`
	StartedAt    time.Time     `json:"started_at"`
	Duration     time.Duration `json:"duration_ns"`
	Status       string        `json:"status"`
	RCode        string        `json:"rcode"`
	Answers      []string      `json:"answers"`
	ErrorKind    string        `json:"error_kind,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

var types = map[string]uint16{"A": 1, "NS": 2, "CNAME": 5, "SOA": 6, "PTR": 12, "MX": 15, "TXT": 16, "AAAA": 28, "SRV": 33, "CAA": 257, "ANY": 255}

func TypeCode(s string) (uint16, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if n, ok := types[s]; ok {
		return n, nil
	}
	if strings.HasPrefix(s, "TYPE") {
		n, err := strconv.ParseUint(strings.TrimPrefix(s, "TYPE"), 10, 16)
		if err == nil {
			return uint16(n), nil
		}
	}
	return 0, fmt.Errorf("unsupported DNS record type %q", s)
}
func TypeName(n uint16) string {
	for s, v := range types {
		if v == n {
			return s
		}
	}
	return "TYPE" + strconv.Itoa(int(n))
}

func Query(ctx context.Context, domain, qtype, resolver string, timeout time.Duration) Result {
	start := time.Now()
	r := Result{Domain: strings.TrimSuffix(domain, "."), QueryType: strings.ToUpper(qtype), Resolver: resolver, Transport: "udp", StartedAt: start, Status: "error", RCode: "-", Answers: []string{}}
	code, err := TypeCode(qtype)
	if err != nil {
		r.ErrorKind = "invalid_query"
		r.ErrorMessage = err.Error()
		r.Duration = time.Since(start)
		return r
	}
	name, err := encodeName(domain)
	if err != nil {
		r.ErrorKind = "invalid_query"
		r.ErrorMessage = err.Error()
		r.Duration = time.Since(start)
		return r
	}
	addr := resolver
	if _, _, e := net.SplitHostPort(addr); e != nil {
		addr = net.JoinHostPort(strings.Trim(addr, "[]"), "53")
	}
	id := uint16(time.Now().UnixNano())
	packet := make([]byte, 12+len(name)+4)
	binary.BigEndian.PutUint16(packet, id)
	binary.BigEndian.PutUint16(packet[2:], 0x0100)
	binary.BigEndian.PutUint16(packet[4:], 1)
	copy(packet[12:], name)
	off := 12 + len(name)
	binary.BigEndian.PutUint16(packet[off:], code)
	binary.BigEndian.PutUint16(packet[off+2:], 1)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, e := (&net.Dialer{}).DialContext(ctx, "udp", addr)
	if e != nil {
		setError(&r, e, ctx)
		r.Duration = time.Since(start)
		return r
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if _, e = conn.Write(packet); e != nil {
		setError(&r, e, ctx)
		r.Duration = time.Since(start)
		return r
	}
	buf := make([]byte, 65535)
	n, e := conn.Read(buf)
	if e != nil {
		setError(&r, e, ctx)
		r.Duration = time.Since(start)
		return r
	}
	if n < 12 {
		r.ErrorKind = "parse_error"
		r.ErrorMessage = "DNS response header is truncated"
		r.Duration = time.Since(start)
		return r
	}
	resp := buf[:n]
	if binary.BigEndian.Uint16(resp) != id {
		r.ErrorKind = "parse_error"
		r.ErrorMessage = "DNS response ID does not match request"
		r.Duration = time.Since(start)
		return r
	}
	flags := binary.BigEndian.Uint16(resp[2:])
	if flags&0x8000 == 0 {
		r.ErrorKind = "parse_error"
		r.ErrorMessage = "packet is not a DNS response"
		r.Duration = time.Since(start)
		return r
	}
	if flags&0x0200 != 0 {
		resp, e = exchangeTCP(ctx, addr, packet)
		if e != nil {
			setError(&r, e, ctx)
			r.Duration = time.Since(start)
			return r
		}
		r.Transport = "tcp"
		if len(resp) < 12 {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "TCP DNS response header is truncated"
			r.Duration = time.Since(start)
			return r
		}
		if binary.BigEndian.Uint16(resp) != id {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "DNS response ID does not match request"
			r.Duration = time.Since(start)
			return r
		}
		flags = binary.BigEndian.Uint16(resp[2:])
		if flags&0x8000 == 0 {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "TCP packet is not a DNS response"
			r.Duration = time.Since(start)
			return r
		}
	}
	r.RCode = rcodeName(flags & 15)
	qcount := int(binary.BigEndian.Uint16(resp[4:]))
	ancount := int(binary.BigEndian.Uint16(resp[6:]))
	off = 12
	for i := 0; i < qcount; i++ {
		_, off, e = decodeName(resp, off)
		if e != nil || off+4 > len(resp) {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "malformed DNS question"
			r.Duration = time.Since(start)
			return r
		}
		off += 4
	}
	for i := 0; i < ancount; i++ {
		nm, next, e := decodeName(resp, off)
		if e != nil || next+10 > len(resp) {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "malformed DNS answer"
			r.Duration = time.Since(start)
			return r
		}
		typ := binary.BigEndian.Uint16(resp[next:])
		class := binary.BigEndian.Uint16(resp[next+2:])
		rdlen := int(binary.BigEndian.Uint16(resp[next+8:]))
		data := next + 10
		end := data + rdlen
		if end > len(resp) {
			r.ErrorKind = "parse_error"
			r.ErrorMessage = "truncated DNS record data"
			r.Duration = time.Since(start)
			return r
		}
		if class == 1 && typ == code {
			ans, e := formatRecord(resp, data, end, typ, nm)
			if e != nil {
				r.ErrorKind = "parse_error"
				r.ErrorMessage = e.Error()
				r.Duration = time.Since(start)
				return r
			}
			r.Answers = append(r.Answers, ans)
		}
		off = end
	}
	if r.RCode == "NOERROR" {
		r.Status = "success"
	} else {
		r.Status = "rcode"
	}
	r.Duration = time.Since(start)
	return r
}
func setError(r *Result, e error, ctx context.Context) {
	var netErr net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(e, &netErr) && netErr.Timeout() {
		r.Status = "timeout"
		r.ErrorKind = "timeout"
		r.ErrorMessage = "query timed out"
	} else if ctx.Err() != nil {
		r.Status = "cancelled"
		r.ErrorKind = "cancelled"
		r.ErrorMessage = ctx.Err().Error()
	} else {
		r.Status = "error"
		r.ErrorKind = "network"
		r.ErrorMessage = e.Error()
	}
}
func encodeName(name string) ([]byte, error) {
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return []byte{0}, nil
	}
	var out []byte
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("invalid DNS name %q", name)
		}
		out = append(out, byte(len(label)))
		out = append(out, []byte(label)...)
		if len(out) > 254 {
			return nil, fmt.Errorf("DNS name is too long")
		}
	}
	return append(out, 0), nil
}
func decodeName(b []byte, off int) (string, int, error) {
	var parts []string
	next := -1
	hops := 0
	for {
		if off >= len(b) {
			return "", 0, fmt.Errorf("name beyond packet")
		}
		n := int(b[off])
		if n&0xc0 == 0xc0 {
			if off+1 >= len(b) {
				return "", 0, fmt.Errorf("bad compression pointer")
			}
			ptr := ((n & 0x3f) << 8) | int(b[off+1])
			if ptr >= len(b) {
				return "", 0, fmt.Errorf("bad compression offset")
			}
			if next < 0 {
				next = off + 2
			}
			off = ptr
			hops++
			if hops > 32 {
				return "", 0, fmt.Errorf("compression loop")
			}
			continue
		}
		off++
		if n == 0 {
			if next < 0 {
				next = off
			}
			break
		}
		if n&0xc0 != 0 || off+n > len(b) {
			return "", 0, fmt.Errorf("invalid label")
		}
		parts = append(parts, string(b[off:off+n]))
		off += n
	}
	return strings.Join(parts, "."), next, nil
}
func formatRecord(b []byte, start, end int, typ uint16, owner string) (string, error) {
	d := b[start:end]
	switch typ {
	case 1:
		if len(d) != 4 {
			return "", fmt.Errorf("invalid A record")
		}
		return net.IP(d).String(), nil
	case 28:
		if len(d) != 16 {
			return "", fmt.Errorf("invalid AAAA record")
		}
		return net.IP(d).String(), nil
	case 2, 5, 12:
		n, _, e := decodeName(b, start)
		return n, e
	case 15:
		if len(d) < 3 {
			return "", fmt.Errorf("invalid MX record")
		}
		n, _, e := decodeName(b, start+2)
		return fmt.Sprintf("%d %s", binary.BigEndian.Uint16(d), n), e
	case 16:
		var xs []string
		for i := 0; i < len(d); {
			l := int(d[i])
			i++
			if i+l > len(d) {
				return "", fmt.Errorf("invalid TXT record")
			}
			xs = append(xs, strconv.Quote(string(d[i:i+l])))
			i += l
		}
		return strings.Join(xs, " "), nil
	case 33:
		if len(d) < 7 {
			return "", fmt.Errorf("invalid SRV record")
		}
		n, _, e := decodeName(b, start+6)
		return fmt.Sprintf("%d %d %d %s", binary.BigEndian.Uint16(d), binary.BigEndian.Uint16(d[2:]), binary.BigEndian.Uint16(d[4:]), n), e
	case 257:
		if len(d) < 2 {
			return "", fmt.Errorf("invalid CAA record")
		}
		l := int(d[1])
		if 2+l > len(d) {
			return "", fmt.Errorf("invalid CAA record")
		}
		return fmt.Sprintf("%d %s %q", d[0], string(d[2:2+l]), string(d[2+l:])), nil
	default:
		return fmt.Sprintf("\\# %d %x", len(d), d), nil
	}
}
func rcodeName(n uint16) string {
	switch n {
	case 0:
		return "NOERROR"
	case 1:
		return "FORMERR"
	case 2:
		return "SERVFAIL"
	case 3:
		return "NXDOMAIN"
	case 4:
		return "NOTIMP"
	case 5:
		return "REFUSED"
	default:
		return fmt.Sprintf("RCODE%d", n)
	}
}

func exchangeTCP(ctx context.Context, addr string, packet []byte) ([]byte, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	framed := make([]byte, 2+len(packet))
	binary.BigEndian.PutUint16(framed, uint16(len(packet)))
	copy(framed[2:], packet)
	if _, err = conn.Write(framed); err != nil {
		return nil, err
	}
	var size [2]byte
	if _, err = io.ReadFull(conn, size[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(size[:]))
	if n < 12 {
		return nil, fmt.Errorf("invalid TCP DNS response length %d", n)
	}
	resp := make([]byte, n)
	if _, err = io.ReadFull(conn, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
