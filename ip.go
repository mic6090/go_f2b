package main

import (
	"errors"
	"fmt"
)

type IPv4 uint32

// parse IPv4 address:
// a.b.c.d
// a.b.c, c < 65536
// a.b,   b < 16777216
// a,     a < 4294967296
func parseIPv4(ip string) (IPv4, error) {
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

/*
func parseIPv4(ip string) IPv4 {
	var ipv4 IPv4 = 0
	for i := 0; i < 4; i++ {
		if len(ip) == 0 {
			return ipv4
		}
		if i > 0 {
			if ip[0] != '.' {
				return ipv4
			}
			ip = ip[1:]
		}
		n, c, ok := dtoi(ip)
		if !ok {
			return ipv4
		}
		ip = ip[c:]
		ipv4 = ipv4<<8 + IPv4(n&0xFF)
	}
	return ipv4
}
*/

func (ip IPv4) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip>>16&0xFF, ip>>8&0xFF, ip&0xFF)
}

/*
func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n > 255 {
			return 255, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}
*/

func indexChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
