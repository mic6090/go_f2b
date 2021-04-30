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
		res, err := parseIPv4(tc.in)
		if err != nil {
			t.Errorf("Unexpected error for input '%s'", tc.in)
		} else if res != tc.out {
			t.Errorf("parseIPv4 for input %q return %s, expected %s", tc.in, res.String(), tc.out.String())
		}
	}
	for _, tc := range ParseIPv4CasesFail {
		res, err := parseIPv4(tc)
		if err == nil {
			t.Errorf("parseIPv4 should fail for input %q, return value %q and success", tc, res.String())
		}
	}
}

func BenchmarkParseIPv4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tc := range ParseIPv4CasesSuccess {
			_, _ = parseIPv4(tc.in)
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
			t.Errorf("IPv4.String for input %q return %s, expected %s", tc.ip, res, tc.res)
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
