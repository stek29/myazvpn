package nftmap

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gammazero/deque"
)

const (
	remapTTL = 300 * time.Second
)

type DNSMapper struct {
	// mu locks both ipPipe and revmap
	mu sync.RWMutex
	// reverse map from public IPv4 to private
	ipMap map[uint32]uint32
	// ipPipe is a deque to allocate IPs from
	ipPipe *deque.Deque[uint32]
	// inner stores the forward mapper this one operates on
	inner NFTMapper
}

func NewDNSMapper(cidr net.IPNet, inner NFTMapper) (*DNSMapper, error) {
	ip := cidr.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("cant get ipv4 from cidr")
	}
	maskOnes, maskSize := cidr.Mask.Size()
	if maskSize != ipv4MaskSize {
		return nil, fmt.Errorf("expected mask of size %d for ipv4, got %v", ipv4MaskSize, maskSize)
	}

	base := ip4ToUint32(ip)
	ipCount := maskOnesToHostCount(maskOnes)

	m := DNSMapper{
		ipMap: make(map[uint32]uint32),
		inner: inner,
	}

	loaded, err := m.loadInner(cidr)
	if err != nil {
		return nil, fmt.Errorf("m.loadInner: %w", err)
	}
	m.fillPipe(base, uint32(ipCount), loaded)

	return &m, nil
}

func (m *DNSMapper) loadInner(cidr net.IPNet) (map[uint32]struct{}, error) {
	elems, err := m.inner.List()
	if err != nil {
		return nil, fmt.Errorf("m.inner.List: %w", err)
	}

	for _, elem := range elems {
		ip := uint32ToIP4(elem.Key)
		if !cidr.Contains(ip) {
			return nil, fmt.Errorf("inner contains %v which is not in cidr %v", ip, cidr)
		}
	}

	loaded := make(map[uint32]struct{})
	m.mu.Lock()
	for _, elem := range elems {
		loaded[elem.Key] = struct{}{}
		m.ipMap[elem.Val] = elem.Key
	}
	m.mu.Unlock()

	return loaded, nil
}

func (m *DNSMapper) fillPipe(base, count uint32, taken map[uint32]struct{}) {
	m.ipPipe = deque.New[uint32](int(count))

	max := base + count

	// optimized variant if ipMap is empty
	if len(taken) == 0 {
		for i := base; i != max; i++ {
			m.ipPipe.PushBack(i)
		}
	} else {
		for i := base; i != max; i++ {
			if _, ok := taken[i]; !ok {
				m.ipPipe.PushBack(i)
			}
		}
	}
}

// RemapIP conforms to IPMapper interface
func (m *DNSMapper) RemapIP(ip net.IP) (net.IP, time.Duration, error) {
	ip4 := ip4ToUint32(ip)

	// check if IP is remapped already
	m.mu.RLock()
	newip4, ok := m.ipMap[ip4]
	m.mu.RUnlock()

	// if not - allocate a new mapping
	if !ok {
		var err error
		newip4, err = m.remapIPNew(ip4)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to set up new remapping: %w", err)
		}
	}

	// return the remapped address
	newIP := uint32ToIP4(newip4)
	return newIP, remapTTL, nil
}

// remapIPNew allocates a new ip mapping for RemapIP
func (m *DNSMapper) remapIPNew(ip4 uint32) (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipPipe.Len() == 0 {
		return 0, fmt.Errorf("no available free IPs to remap to")
	}

	newip4 := m.ipPipe.PopFront()

	err := m.inner.Add(NFTMapElem{Key: newip4, Val: ip4})
	if err != nil {
		// put back since Add failed
		m.ipPipe.PushFront(newip4)
		return 0, fmt.Errorf("inner.Add: %w", err)
	}

	m.ipMap[ip4] = newip4
	return newip4, nil
}
