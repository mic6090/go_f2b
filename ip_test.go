package main

import (
	"testing"
)

var ParseIPv4CasesSuccess = []struct {
	in  string
	out IPv4
}{
	{in: "0.0.0.0", out: 0},
	{in: "1", out: 1},
	{in: "1.2", out: 16777218},
	{in: "15.30.65521", out: 253689841},
	{in: "1.16777215", out: 33554431},
	{in: "255.255.255.255", out: 4294967295},
	{in: "192.168.1.129", out: 3232235905},
}
var ParseIPv4CasesFail = []string{
	"", "1.16777216", "10.12.65537", "3.6.9.257", "3.6.9,257", "3.258.9.255", "4.7.", "4294967297", "15.4294967297", "3.6.9.255.",
}

func TestParseIPv4(t *testing.T) {
	for _, tc := range ParseIPv4CasesSuccess {
		res, err := ParseIPv4(tc.in)
		if err != nil {
			t.Errorf("Unexpected error for input '%s'", tc.in)
		} else if res != tc.out {
			t.Errorf("ParseIPv4 for input %q return %s, expected %s", tc.in, res.String(), tc.out.String())
		}
	}
	for _, tc := range ParseIPv4CasesFail {
		res, err := ParseIPv4(tc)
		if err == nil {
			t.Errorf("ParseIPv4 should fail for input %q, return value %q and success", tc, res.String())
		}
	}
}

func BenchmarkParseIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tc := range ParseIPv4CasesSuccess {
			_, _ = ParseIPv4(tc.in)
		}
	}
}

var ParseCIDRCases = []struct {
	in   string
	ip   IPv4
	net  IPv4
	mask IPMask
	fail bool
}{
	{"192.168.1.129/28", 0xc0a80181, 0xc0a80180, 0xfffffff0, false},
	{"192.168.1.130/24", 0xc0a80182, 0xc0a80100, 0xffffff00, false},
	{"192.168.1.127", 0xc0a8017f, 0xc0a8017f, 0xffffffff, false},
	{"192.168.1.129/33", 0, 0, 0, true},
	{"95.186.65538/24", 0, 0, 0, true},
	{"2/294", 0, 0, 0, true},
	{"5/-24", 0, 0, 0, true},
	{"7.7/1:6", 0, 0, 0, true},
}

func TestParseCIDR(t *testing.T) {
	for _, tc := range ParseIPv4CasesSuccess {
		ip, net, err := ParseCIDR(tc.in)
		if err != nil {
			t.Errorf("Unexpected error for input %q", tc.in)
			continue
		}
		if ip != tc.out || net.IP != tc.out || net.Mask != 0xffffffff {
			t.Errorf("ParseCIDR error for input %q", tc.in)
		}
	}

	for _, tc := range ParseCIDRCases {
		ip, net, err := ParseCIDR(tc.in)
		if (err != nil) != tc.fail {
			if err != nil {
				t.Errorf("Unexpected error for input %q", tc.in)
			} else {
				t.Errorf("Unexpected success for input %q", tc.in)
			}
			continue
		}
		if net == nil {
			if !tc.fail {
				t.Errorf("ParseCIDR error for input %q", tc.in)
			}
			continue
		}
		if ip != tc.ip || net.IP != tc.net || net.Mask != tc.mask {
			t.Errorf("ParseCIDR error for input %q", tc.in)
		}
	}
}

var IPv4MaskCases = []struct {
	a, b, c, d byte
	res        IPMask
}{
	{1, 2, 3, 4, 0x01020304},
	{4, 3, 2, 1, 0x04030201},
	{255, 254, 253, 252, 0xfffefdfc},
}

func TestIPv4Mask(t *testing.T) {
	for _, tc := range IPv4MaskCases {
		res := IPv4Mask(tc.a, tc.b, tc.c, tc.d)
		if res != tc.res {
			t.Errorf("IPv4Mask for input %q returns %x, expected %x",
				[]byte{tc.a, tc.b, tc.c, tc.d}, res, tc.res)
		}
	}
}

var IPv4StringCases = []struct {
	ip  IPv4
	res string
}{
	{ip: 1, res: "0.0.0.1"},
	{ip: 257, res: "0.0.1.1"},
	{ip: 65537, res: "0.1.0.1"},
}

func TestIPv4String(t *testing.T) {
	for _, tc := range IPv4StringCases {
		res := tc.ip.String()
		if tc.res != res {
			t.Errorf("IPv4.String for input %q returns %s, expected %s", tc.ip, res, tc.res)
		}
	}
}

var IPMaskCases = []struct {
	len  int
	mask IPMask
}{
	{len: -1, mask: 0xffffffff},
	{len: 0, mask: 0},
	{len: 1, mask: 0x80000000},
	{len: 2, mask: 0xc0000000},
	{len: 8, mask: 0xff000000},
	{len: 16, mask: 0xffff0000},
	{len: 24, mask: 0xffffff00},
	{len: 27, mask: 0xffffffe0},
	{len: 31, mask: 0xfffffffe},
	{len: 32, mask: 0xffffffff},
	{len: 99, mask: 0xffffffff},
}

var IPMaskCasesFail = []IPMask{
	0x80000010, 0x80000200,
	0xf7000000, 0xfeffffff,
	0x0fffffff, 0xff8fffff,
}

func TestCIDRMask(t *testing.T) {
	for _, tc := range IPMaskCases {
		res := CIDRMask(tc.len)
		if tc.mask != res {
			t.Errorf("CIDRMask for input %d returns %q, expected %q", tc.len, res, tc.mask)
		}
	}
}

func TestIPMask_Size(t *testing.T) {
	for _, tc := range IPMaskCases {
		if 0 <= tc.len && tc.len <= 32 {
			size := tc.mask.Size()
			if tc.len != size {
				t.Errorf("IPMask.Size for input %q return %d, expected %d",
					tc.mask.String(), size, tc.len)
			}
		}
	}
	for _, tc := range IPMaskCasesFail {
		res := tc.Size()
		if res != -1 {
			t.Errorf("IPMask.Size for input %q return %d, expected -1 (error)", tc.String(), res)
		}
	}
}

var IPMaskStringCases = []struct {
	mask IPMask
	s    string
}{
	{mask: 0, s: "00000000"},
	{mask: 0x80000000, s: "80000000"},
	{mask: 0xc0000000, s: "c0000000"},
	{mask: 0xe0000000, s: "e0000000"},
	{mask: 0xf0000000, s: "f0000000"},
	{mask: 0xffe00000, s: "ffe00000"},
	{mask: 0xfffc0000, s: "fffc0000"},
	{mask: 0xfffffffc, s: "fffffffc"},
}

func TestIPMask_String(t *testing.T) {
	for _, tc := range IPMaskStringCases {
		res := tc.mask.String()
		if res != tc.s {
			t.Errorf("IPMask.String for input %q return %s, expected %s",
				tc.mask, res, tc.s)
		}
	}
}

var MakeIPNetCases = []struct {
	ip   IPv4
	net  IPv4
	mask IPMask
}{
	{0xc0a80181, 0xc0a80180, 0xfffffff0},
	{0xc0a80181, 0xc0a80100, 0xffffff00},
	{0x08080808, 0x08000000, 0xff000000},
}

func TestMakeIPNet(t *testing.T) {
	for _, tc := range MakeIPNetCases {
		res := MakeIPNet(tc.ip, tc.mask)
		if res.IP != tc.net || res.Mask != tc.mask {
			t.Errorf("MakeIPNet for input {%q %q} return %q, expected {%q %q}",
				tc.ip, tc.mask, *res, tc.net, tc.mask)
		}
	}
}

var indexCharCases = []struct {
	s   string
	c   byte
	res int
}{
	{s: "1.2.3.4", c: '.', res: 1},
	{s: "1.2.3.4", c: ',', res: -1},
	{s: "   ", c: ' ', res: 0},
	{s: "", c: ',', res: -1},
}

func TestIndexChar(t *testing.T) {
	for _, tc := range indexCharCases {
		res := indexChar(tc.s, tc.c)
		if res != tc.res {
			t.Errorf("indexChar for input {s=%q,c='%c'} return %d, expected %d.", tc.s, tc.c, res, tc.res)
		}
	}
}
