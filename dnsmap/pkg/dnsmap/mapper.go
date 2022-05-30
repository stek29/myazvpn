package dnsmap

import (
	"net"
	"time"
)

// IPMapper remaps provided net.IP to a different net.IP
// notice: it must not alter given net.IP in any way
// notice: returned net.IP also should not be modified
type IPMapper interface {
	RemapIP(net.IP) (net.IP, time.Duration, error)
}

// NoopMapper returns original addresses – in other words,
// it remaps addresses to themselves
type NoopMapper struct{}

// RemapIP conforms to IPMapper interface
func (NoopMapper) RemapIP(ip net.IP) (net.IP, time.Duration, error) {
	return ip, 0, nil
}
