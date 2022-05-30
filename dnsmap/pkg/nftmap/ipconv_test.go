package nftmap

import (
	"net"
	"testing"
)

func TestMaskOnesToHostCount(t *testing.T) {
	cases := []struct{ mask, count int }{
		{32, 1},
		{31, 2},
		{30, 4},
		{24, 256},
		{16, 65536},
		{0, 4294967296},
	}

	for _, tc := range cases {
		if got := maskOnesToHostCount(tc.mask); got != tc.count {
			t.Errorf("maskOnesToHostCount(%d): expected %d, got %d", tc.mask, tc.count, got)
		}
	}
}

func TestIP4ToUint32(t *testing.T) {
	cases := []struct {
		ip  net.IP
		val uint32
	}{
		{net.ParseIP("1.1.1.1"), 0x01010101},
		{net.ParseIP("1.2.3.4"), 0x01020304},
		{net.ParseIP("255.255.255.255"), 0xffffffff},
	}

	for _, tc := range cases {
		if got := ip4ToUint32(tc.ip); got != tc.val {
			t.Errorf("ip4ToUint32(%d): expected 0x%x, got 0x%x", tc.ip, tc.val, got)
		}
	}
}

func TestUint32ToIP4(t *testing.T) {
	cases := []struct {
		ip  net.IP
		val uint32
	}{
		{net.ParseIP("1.1.1.1"), 0x01010101},
		{net.ParseIP("1.2.3.4"), 0x01020304},
		{net.ParseIP("255.255.255.255"), 0xffffffff},
	}

	for _, tc := range cases {
		if got := uint32ToIP4(tc.val); !tc.ip.Equal(got) {
			t.Errorf("uint32ToIP4(0x%x): expected %v, got %v", tc.val, tc.ip, got)
		}
	}
}
