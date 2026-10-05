// Package safeurl validates user-supplied URLs to reduce SSRF risk.
package safeurl

import (
	"context"
	"errors"
	"net"
	"net/url"
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
