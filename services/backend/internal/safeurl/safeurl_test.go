package safeurl

import (
	"context"
	"net"
	"testing"
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
