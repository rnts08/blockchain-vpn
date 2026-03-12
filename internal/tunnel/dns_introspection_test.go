package tunnel

import "testing"

func TestHasExpectedSecureDNS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		servers  []string
		expected bool
	}{
		{
			name:     "empty servers",
			servers:  []string{},
			expected: false,
		},
		{
			name:     "cloudflare only",
			servers:  []string{"1.1.1.1"},
			expected: true,
		},
		{
			name:     "google only",
			servers:  []string{"8.8.8.8"},
			expected: true,
		},
		{
			name:     "both secure servers",
			servers:  []string{"1.1.1.1", "8.8.8.8"},
			expected: true,
		},
		{
			name:     "insecure server only",
			servers:  []string{"192.168.1.1"},
			expected: false,
		},
		{
			name:     "mixed servers",
			servers:  []string{"192.168.1.1", "1.1.1.1", "10.0.0.1"},
			expected: true,
		},
		{
			name:     "with whitespace",
			servers:  []string{"  1.1.1.1  "},
			expected: true,
		},
		{
			name:     "ipv6 servers ignored",
			servers:  []string{"::1", "fe80::1"},
			expected: false,
		},
		{
			name:     "localhost ignored",
			servers:  []string{"127.0.0.1"},
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasExpectedSecureDNS(tc.servers)
			if result != tc.expected {
				t.Errorf("hasExpectedSecureDNS(%v) = %v, want %v", tc.servers, result, tc.expected)
			}
		})
	}
}
