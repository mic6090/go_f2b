package main

import (
	"errors"
	"fmt"
	"strconv"
)

type IPv4 uint32

// parse IPv4 address:
// a.b.c.d
// a.b.c, c < 65536
// a.b,   b < 16777216
// a,     a < 4294967296
func ParseIPv4(ip string) (IPv4, error) {
	if len(ip) == 0 {
		return 0, errors.New("empty string")
	}

	var ipv4 [4]uint32
	var maxo uint32 = 0xffffffff
	var n uint64
	var i, k int
	for i = 0; i < 4; i++ {
		n = 0
		for k = 0; k < len(ip) && '0' <= ip[k] && ip[k] <= '9'; k++ {
			n = n*10 + uint64(ip[k]-'0')
			if n > uint64(maxo) {
				return 0, errors.New("too large octet value")
			}
		}
		if k == 0 {
			return 0, errors.New("empty octet")
		}
		ipv4[i] = uint32(n)
		ip = ip[k:]
		if len(ip) == 0 {
			break
		}
		if ip[0] != '.' {
			return 0, errors.New("wrong symbol")
		}
		ip = ip[1:]
	}
	if i == 4 {
		return 0, errors.New("extra symbols")
	}
	if i == 0 {
		return IPv4(ipv4[0]), nil
	}
	var res IPv4
	for j := 0; j < i; j++ {
		if ipv4[j] > 255 {
			return 0, errors.New("too large octet value")
		}
		res = res<<8 + IPv4(ipv4[j])
	}
	if ipv4[i] > uint32(maxo>>(i*8)) {
		return 0, errors.New("too large octet value")
	}
	res = res<<(32-i*8) + IPv4(ipv4[i])

	return res, nil
}

func ParseCIDR(s string) (IPv4, *IPNet, error) {
	pos := indexChar(s, '/')
	var addr, mask string
	if pos < 0 { // address as /32 subnet
		addr, mask = s, ""
	} else {
		addr, mask = s[:pos], s[pos+1:]
	}
	ip, err := ParseIPv4(addr)
	if err != nil {
		return 0, nil, err
	}
	var k, n int
	for k = 0; k < len(mask) && '0' <= mask[k] && mask[k] <= '9'; k++ {
		n = n*10 + int(mask[k]-'0')
		if n > 32 {
			return 0, nil, errors.New("too large mask value")
		}
	}
	if k != len(mask) {
		return 0, nil, errors.New("bad mask value")
	}
	if k == 0 {
		n = 32
	}
	m := CIDRMask(n)
	return ip, &IPNet{ip.Mask(m), m}, nil
}

func (ip IPv4) Mask(m IPMask) IPv4 {
	return ip & IPv4(m)
}

func (ip IPv4) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip>>16&0xFF, ip>>8&0xFF, ip&0xFF)
}

type IPMask uint32

func CIDRMask(ones int) IPMask {
	m := IPMask(^uint32(0))
	if 0 > ones || ones > 32 {
		return m
	}
	return m << (32 - ones)
}

func IPv4Mask(a, b, c, d byte) IPMask {
	return IPMask(a)<<24 | IPMask(b)<<16 | IPMask(c)<<8 | IPMask(d)
}

func (m IPMask) Size() int {
	var n int
	v := m
	for v&0x80000000 != 0 {
		n++
		v <<= 1
	}
	if v != 0 {
		return -1
	}
	return n
}

func (m IPMask) String() string {
	return fmt.Sprintf("%08x", uint32(m))
}

type IPNet struct {
	IP   IPv4
	Mask IPMask
}

func MakeIPNet(ip IPv4, mask IPMask) *IPNet {
	return &IPNet{IP: ip.Mask(mask), Mask: mask}
}

func (n *IPNet) Contains(ip IPv4) bool {
	return ip.Mask(n.Mask) == n.IP
}

func (n *IPNet) String() string {
	l := n.Mask.Size()
	if l < 0 {
		return n.IP.String() + "/" + n.Mask.String()
	}
	return n.IP.String() + "/" + strconv.Itoa(l)
}

func indexChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
