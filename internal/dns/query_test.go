package dns

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func testResolver(t *testing.T, mode string) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback sockets unavailable in this environment: %v", err)
	}
	addr := pc.LocalAddr().String()
	go func() {
		defer pc.Close()
		buf := make([]byte, 2048)
		n, peer, e := pc.ReadFrom(buf)
		if e != nil {
			return
		}
		q := append([]byte(nil), buf[:n]...)
		if mode == "timeout" {
			return
		}
		response := testResponse(q, mode)
		_, _ = pc.WriteTo(response, peer)
	}()
	return addr
}
func testResponse(q []byte, mode string) []byte {
	r := append([]byte(nil), q...)
	flags := uint16(0x8180)
	answers := uint16(1)
	if mode == "nxdomain" {
		flags = 0x8183
		answers = 0
	}
	if mode == "truncated" {
		flags |= 0x0200
		answers = 0
	}
	binary.BigEndian.PutUint16(r[2:], flags)
	binary.BigEndian.PutUint16(r[6:], answers)
	if answers > 0 {
		r = append(r, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 30, 0, 4, 192, 0, 2, 42)
	}
	return r
}
func TestQueryControlledResolver(t *testing.T) {
	addr := testResolver(t, "success")
	got := Query(context.Background(), "sample.test", "A", addr, time.Second)
	if got.Status != "success" || got.RCode != "NOERROR" || len(got.Answers) != 1 || got.Answers[0] != "192.0.2.42" {
		t.Fatalf("unexpected result: %+v", got)
	}
}
func TestQueryDistinguishesNXDOMAIN(t *testing.T) {
	addr := testResolver(t, "nxdomain")
	got := Query(context.Background(), "missing.test", "A", addr, time.Second)
	if got.Status != "rcode" || got.RCode != "NXDOMAIN" {
		t.Fatalf("unexpected result: %+v", got)
	}
}
func TestQueryTimeout(t *testing.T) {
	addr := testResolver(t, "timeout")
	got := Query(context.Background(), "silent.test", "A", addr, 40*time.Millisecond)
	if got.Status != "timeout" {
		t.Fatalf("status=%s result=%+v", got.Status, got)
	}
}
func TestTypeCodeRejectsUnknown(t *testing.T) {
	if _, err := TypeCode("BOGUS"); err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestFormatARecord(t *testing.T) {
	got, err := formatRecord([]byte{192, 0, 2, 7}, 0, 4, 1, "example.test")
	if err != nil || got != "192.0.2.7" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestEncodeDecodeName(t *testing.T) {
	wire, err := encodeName("api.example.test.")
	if err != nil {
		t.Fatal(err)
	}
	got, next, err := decodeName(wire, 0)
	if err != nil || got != "api.example.test" || next != len(wire) {
		t.Fatalf("got %q next %d err %v", got, next, err)
	}
}

func TestQueryRetriesTruncatedUDPOverTCP(t *testing.T) {
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback sockets unavailable in this environment: %v", err)
	}
	defer tcp.Close()
	pc, err := net.ListenPacket("udp", tcp.Addr().String())
	if err != nil {
		t.Skipf("loopback UDP unavailable in this environment: %v", err)
	}
	defer pc.Close()
	go func() {
		buf := make([]byte, 2048)
		n, peer, e := pc.ReadFrom(buf)
		if e != nil {
			return
		}
		_, _ = pc.WriteTo(testResponse(buf[:n], "truncated"), peer)
	}()
	go func() {
		conn, e := tcp.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		var size [2]byte
		if _, e = io.ReadFull(conn, size[:]); e != nil {
			return
		}
		q := make([]byte, int(binary.BigEndian.Uint16(size[:])))
		if _, e = io.ReadFull(conn, q); e != nil {
			return
		}
		resp := testResponse(q, "success")
		var frame [2]byte
		binary.BigEndian.PutUint16(frame[:], uint16(len(resp)))
		_, _ = conn.Write(frame[:])
		_, _ = conn.Write(resp)
	}()
	got := Query(context.Background(), "sample.test", "A", tcp.Addr().String(), time.Second)
	if got.Status != "success" || got.Transport != "tcp" || len(got.Answers) != 1 {
		t.Fatalf("unexpected TCP fallback result: %+v", got)
	}
}
