package common

import (
	"strings"
	"testing"
)

func TestValidateURLRejectsSpecialPurposeAddresses(t *testing.T) {
	tests := []string{
		"http://0.0.0.0/",
		"http://100.64.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://192.0.2.10/",
		"http://198.18.0.1/",
		"http://203.0.113.10/",
		"http://224.0.0.1/",
		"http://240.0.0.1/",
		"http://[::1]/",
		"http://[::ffff:127.0.0.1]/",
		"http://[64:ff9b::808:808]/",
		"http://[100::1]/",
		"http://[2001:db8::1]/",
		"http://[fc00::1]/",
		"http://[fe80::1]/",
		"http://[ff02::1]/",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			err := ValidateURLWithFetchSetting(url, true, false, false, false, nil, nil, nil, false)
			if err == nil {
				t.Fatalf("ValidateURLWithFetchSetting(%q) accepted a special-purpose address", url)
			}
			if !strings.Contains(err.Error(), "private IP address not allowed") {
				t.Fatalf("error = %q, want private IP rejection", err.Error())
			}
		})
	}
}

func TestValidateURLStillAllowsPublicAddresses(t *testing.T) {
	err := ValidateURLWithFetchSetting("https://8.8.8.8/dns-query", true, false, false, false, nil, nil, []string{"443"}, false)
	if err != nil {
		t.Fatalf("ValidateURLWithFetchSetting rejected a public HTTPS endpoint: %v", err)
	}
}

func TestValidateURLAppliesIPFilterToResolvedDomains(t *testing.T) {
	err := ValidateURLWithFetchSetting("http://localhost/callback", true, false, false, false, nil, nil, nil, true)
	if err == nil {
		t.Fatal("ValidateURLWithFetchSetting accepted localhost when domain IP filtering is enabled")
	}
	if !strings.Contains(err.Error(), "private IP address not allowed") {
		t.Fatalf("error = %q, want private IP rejection", err.Error())
	}
}
