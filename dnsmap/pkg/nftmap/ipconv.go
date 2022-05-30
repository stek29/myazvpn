package nftmap

import (
	"net"
)

const (
	ipv4MaskSize = 4 * 8
)

func maskOnesToHostCount(maskLen int) int {
	maskZeros := ipv4MaskSize - maskLen
	hosts := 1
	for i := 0; i < maskZeros; i++ {
		hosts *= 2
	}
	return hosts
}

func ip4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		panic("ip4ToUint32 got nil ip.To4")
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP4(v uint32) net.IP {
	return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}
