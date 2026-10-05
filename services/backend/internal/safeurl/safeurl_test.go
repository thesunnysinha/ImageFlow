package safeurl

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBlocked(t *testing.T) {
	for ip, want := range map[string]bool{
		"127.0.0.1": true, "10.0.0.5": true, "192.168.1.1": true,
		"169.254.169.254": true, "::1": true, "0.0.0.0": true,
		"8.8.8.8": false, "1.1.1.1": false,
	} {
		if got := Blocked(net.ParseIP(ip)); got != want {
			t.Errorf("Blocked(%s)=%v want %v", ip, got, want)
		}
	}
}

func TestValidateRejectsBadSchemes(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "ftp://x.com/a", "http://", "://bad", "http://127.0.0.1/x"} {
		if err := Validate(context.Background(), u, nil); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}
}

func TestClientRefusesPrivateAddressesAtDialTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("secret")) }))
	defer srv.Close() // listens on 127.0.0.1

	if _, err := NewClient(2*time.Second, false, true).Get(srv.URL); err == nil {
		t.Fatal("a loopback server must be unreachable through the safe client")
	}
	resp, err := NewClient(2*time.Second, true, true).Get(srv.URL)
	if err != nil {
		t.Fatalf("allowPrivate should connect: %v", err)
	}
	resp.Body.Close()
}

func TestClientDoesNotFollowRedirectsWhenDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			_, _ = w.Write([]byte("end"))
			return
		}
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer srv.Close()
	resp, err := NewClient(2*time.Second, true, false).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("got %d, want the redirect itself (302)", resp.StatusCode)
	}
}
