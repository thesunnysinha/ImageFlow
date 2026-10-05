package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPutOpenRoundTrip(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n, err := l.Put(context.Background(), "job/0.jpg", strings.NewReader("hello"))
	if err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	rc, err := l.Open(context.Background(), "job/0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	if b, _ := io.ReadAll(rc); string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
	if _, err := l.Open(context.Background(), "job/missing"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestKeysCannotEscapeTheRoot(t *testing.T) {
	root := t.TempDir()
	l, _ := NewLocal(filepath.Join(root, "store"))
	if _, err := l.Put(context.Background(), "../escaped", strings.NewReader("x")); err != nil {
		t.Fatalf("a traversal key is anchored inside the root, not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "escaped")); err == nil {
		t.Fatal("object escaped the storage root")
	}
	for _, bad := range []string{"", "a/", "/", "a\x00b"} {
		if _, err := l.Put(context.Background(), bad, strings.NewReader("x")); err == nil {
			t.Errorf("key %q should be rejected", bad)
		}
	}
}
