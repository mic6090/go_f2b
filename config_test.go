package main

import "testing"

var ParseIPListCases = []struct {
	in  string
	res []IPNet
}{
	{"127.0.0.1 192.168.0.0/24", []IPNet{IPNet{0x7f000001, 0xffffffff}, IPNet{0xc0a80000, 0xffffff00}}},
	{"127.0.0.1,192.168.0.0/24", []IPNet{IPNet{0x7f000001, 0xffffffff}, IPNet{0xc0a80000, 0xffffff00}}},
	{"127.0.0.1, 192.168.0.0/24", []IPNet{IPNet{0x7f000001, 0xffffffff}, IPNet{0xc0a80000, 0xffffff00}}},
	{", 127.0.0.1, 192.168.0.0/24, ", []IPNet{IPNet{0x7f000001, 0xffffffff}, IPNet{0xc0a80000, 0xffffff00}}},
}

func TestParseIPList(t *testing.T) {
	for _, tc := range ParseIPListCases {
		res, err := parseIPList(tc.in)
		if err != nil {
			t.Errorf("Unexpected error for input %q", tc.in)
		}
		if len(res) != len(tc.res) {
			t.Errorf("Unexpected error for input %q", tc.in)
		}
		for i := range res {
			if res[i] != tc.res[i] {
				t.Errorf("Unexpected error for input %q", tc.in)
			}
		}
	}
}
