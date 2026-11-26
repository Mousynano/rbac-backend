// Package emailpolicy normalizes and checks email addresses at the platform boundary.
package emailpolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	ErrInvalidAddress = errors.New("invalid email address")
	ErrNoMailDomain   = errors.New("email domain cannot receive mail")
	ErrUnavailable    = errors.New("email domain check unavailable")
)

// Resolver contains only the DNS lookups required by email eligibility.
type Resolver interface {
	LookupMX(context.Context, string) ([]*net.MX, error)
	LookupHost(context.Context, string) ([]string, error)
}

// SystemResolver performs DNS lookups using Go's resolver.
type SystemResolver struct{}

func (SystemResolver) LookupMX(ctx context.Context, host string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, host)
}
func (SystemResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// Normalize validates the supported bare ASCII address form and lowercases it.
func Normalize(address string) (string, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 || len(address) > 254 || !isASCII(address) || strings.Count(address, "@") != 1 || strings.ContainsAny(address, " <>(),:\\\"[]") {
		return "", ErrInvalidAddress
	}
	parts := strings.SplitN(address, "@", 2)
	local, domain := parts[0], strings.ToLower(parts[1])
	if len(local) == 0 || len(local) > 64 || len(domain) == 0 || len(domain) > 253 || strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return "", ErrInvalidAddress
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidAddress
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", ErrInvalidAddress
			}
		}
	}
	for _, c := range local {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune(".!#$%&'*+/=?^_`{|}~-", c)) {
			return "", ErrInvalidAddress
		}
	}
	return strings.ToLower(local) + "@" + domain, nil
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}

// CheckDomain verifies MX or fallback address records and rejects Null MX.
func CheckDomain(ctx context.Context, resolver Resolver, domain string) error {
	records, err := resolver.LookupMX(ctx, domain)
	if err == nil {
		if len(records) > 0 {
			if len(records) == 1 && records[0].Host == "." {
				return ErrNoMailDomain
			}
			return nil
		}
	} else if !isNotFound(err) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	hosts, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		if isNotFound(err) {
			return ErrNoMailDomain
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(hosts) == 0 {
		return ErrNoMailDomain
	}
	return nil
}

func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

// MatchesDomain reports exact or label-boundary subdomain matches.
func MatchesDomain(domain, blocked string) bool {
	domain, blocked = strings.ToLower(strings.TrimSuffix(domain, ".")), strings.ToLower(strings.TrimSuffix(blocked, "."))
	return domain == blocked || strings.HasSuffix(domain, "."+blocked)
}
