package daemon

import (
	"context"
	"fmt"
	"net"
	"time"
)

var defaultProbeEndpoints = []string{
	"1.1.1.1:443",
	"8.8.8.8:53",
	"9.9.9.9:53",
}

// ConnectivityOptions configures TCP endpoint dialing and DNS resolution checks.
type ConnectivityOptions struct {
	Endpoints   []string
	DNSHost     string
	Timeout     time.Duration
	Interval    time.Duration
	DialTimeout time.Duration
	LookupHost  func(ctx context.Context, host string) ([]string, error)
}

// DefaultConnectivityOptions returns production connectivity probe defaults (120s timeout).
func DefaultConnectivityOptions(timeout time.Duration) ConnectivityOptions {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return ConnectivityOptions{
		Endpoints:   append([]string{}, defaultProbeEndpoints...),
		DNSHost:     "dns.google",
		Timeout:     timeout,
		Interval:    1500 * time.Millisecond,
		DialTimeout: 2 * time.Second,
	}
}

// WaitForConnectivity blocks until TCP reachability and DNS resolution succeed or timeout expires.
func WaitForConnectivity(ctx context.Context, timeout time.Duration) error {
	return WaitForConnectivityWithOptions(ctx, DefaultConnectivityOptions(timeout))
}

// WaitForConnectivityWithOptions probes TCP endpoints and DNS resolution with reusable timers and context cancellation.
func WaitForConnectivityWithOptions(ctx context.Context, opts ConnectivityOptions) error {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}
	dialTimeout := opts.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 2 * time.Second
	}
	endpoints := opts.Endpoints
	if len(endpoints) == 0 {
		endpoints = defaultProbeEndpoints
	}
	lookup := opts.LookupHost
	if lookup == nil {
		lookup = net.DefaultResolver.LookupHost
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := net.Dialer{Timeout: dialTimeout}

	tryOnce := func() bool {
		tcpOK := false
		for _, ep := range endpoints {
			conn, err := dialer.DialContext(probeCtx, "tcp", ep)
			if err == nil {
				_ = conn.Close()
				tcpOK = true
				break
			}
		}
		if !tcpOK {
			return false
		}
		if opts.DNSHost == "" {
			return true
		}
		dnsCtx, dnsCancel := context.WithTimeout(probeCtx, dialTimeout)
		defer dnsCancel()
		addrs, err := lookup(dnsCtx, opts.DNSHost)
		return err == nil && len(addrs) > 0
	}

	if tryOnce() {
		return nil
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-probeCtx.Done():
			return fmt.Errorf("network connectivity unavailable after %v: %w", timeout, probeCtx.Err())
		case <-timer.C:
			if tryOnce() {
				return nil
			}
			timer.Reset(interval)
		}
	}
}
