// Package safeurl validates user-supplied URLs to reduce SSRF risk.
package safeurl

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

var ErrBlocked = errors.New("url is not allowed")

// Validate accepts only http(s) URLs whose host does not resolve to a
// private, loopback, link-local or unspecified address.
// NOTE: callers that fetch the URL must also pin the resolved IP at dial
// time (DNS rebinding); this is a first line of defence at the API edge.
func Validate(ctx context.Context, raw string, r *net.Resolver) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return ErrBlocked
	}
	if r == nil {
		r = net.DefaultResolver
	}
	ips, err := r.LookupIPAddr(ctx, u.Hostname())
	if err != nil || len(ips) == 0 {
		return ErrBlocked
	}
	for _, ip := range ips {
		if Blocked(ip.IP) {
			return ErrBlocked
		}
	}
	return nil
}

func Blocked(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// NewClient returns an HTTP client that refuses to connect to private, loopback or link-local
// addresses. The check runs on the resolved address at dial time, so DNS rebinding and
// redirects to internal hosts are blocked too. allowPrivate exists for tests only.
func NewClient(timeout time.Duration, allowPrivate bool, followRedirects bool) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return ErrBlocked
			}
			ip := net.ParseIP(host)
			if ip == nil || (!allowPrivate && Blocked(ip)) {
				return ErrBlocked
			}
			return nil
		},
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
		// Never use proxies from the environment: they would bypass the dial-time check.
		Proxy: nil,
	}
	client := &http.Client{Timeout: timeout, Transport: transport}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !followRedirects {
			return http.ErrUseLastResponse
		}
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return ErrBlocked
		}
		return nil
	}
	return client
}
