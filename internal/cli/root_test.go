package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

func TestIntermixFlagsAfterDomain(t *testing.T) {
	got := intermix([]string{"example.com", "--format", "json", "--resolver=1.1.1.1,8.8.8.8", "--timeout=2s"})
	want := []string{"--format", "json", "--resolver=1.1.1.1,8.8.8.8", "--timeout=2s", "example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
func TestHelpDoesNotNeedNetwork(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := Run(context.Background(), []string{"check", "--help"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 || errOut.Len() != 0 {
		t.Fatalf("out=%q stderr=%q", out.String(), errOut.String())
	}
}
func TestInvalidFormatRejectedBeforeQuery(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Run(context.Background(), []string{"check", "example.com", "--format", "xml"}, &out, &errOut)
	if err == nil {
		t.Fatal("expected invalid format error")
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
