package gateway

import (
	"testing"
)

func TestParseCidr(t *testing.T) {
	type parseCIDRTestCase struct {
		CIDR        string
		expectBegin string
		expectEnd   string
	}
	var testCase = []parseCIDRTestCase{
		{
			CIDR:        "192.168.10.0/24",
			expectBegin: "192.168.10.1",
			expectEnd:   "192.168.10.255",
		},
		{
			CIDR:        "10.1.0.0/8",
			expectBegin: "10.1.0.1",
			expectEnd:   "10.255.255.255",
		},
		{
			CIDR:        "10.2.0.0/8",
			expectBegin: "10.2.0.1",
			expectEnd:   "10.255.255.255",
		},
	}

	for _, tc := range testCase {
		ibegin, iend, err := parseCIDR(tc.CIDR)
		if err != nil {
			t.Error(err)
			return
		}

		if toIP(ibegin) != tc.expectBegin || toIP(iend) != tc.expectEnd {
			t.Errorf("expect begin: %s, end: %s, got begin: %s, end: %s", tc.expectBegin, tc.expectEnd, toIP(ibegin), toIP(iend))
		}
	}
}
