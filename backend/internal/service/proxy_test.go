package service

import (
	"net/url"
	"strings"
	"testing"
)

func TestProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		proxy Proxy
		want  string
	}{
		{
			name: "without auth",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
			},
			want: "http://proxy.example.com:8080",
		},
		{
			name: "with auth",
			proxy: Proxy{
				Protocol: "socks5",
				Host:     "socks.example.com",
				Port:     1080,
				Username: "user",
				Password: "pass",
			},
			want: "socks5://user:pass@socks.example.com:1080",
		},
		{
			name: "username only keeps no auth for compatibility",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
				Username: "user-only",
			},
			want: "http://proxy.example.com:8080",
		},
		{
			name: "with special characters in credentials",
			proxy: Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     3128,
				Username: "first last@corp",
				Password: "p@ ss:#word",
			},
			want: "http://first%20last%40corp:p%40%20ss%3A%23word@proxy.example.com:3128",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.proxy.URL(); got != tc.want {
				t.Fatalf("Proxy.URL() mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestProxyURL_Chained(t *testing.T) {
	upstream := &Proxy{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     7897,
	}
	current := &Proxy{
		Protocol:      "http",
		Host:          "45.56.171.66",
		Port:          7567,
		Username:      "user",
		Password:      "pass",
		UpstreamProxy: upstream,
	}

	got := current.URL()
	if !strings.HasPrefix(got, "chain://proxy?") {
		t.Fatalf("Proxy.URL() chain scheme mismatch: got=%q", got)
	}
	if !strings.Contains(got, "upstream=http%3A%2F%2F127.0.0.1%3A7897") {
		t.Fatalf("Proxy.URL() missing upstream: got=%q", got)
	}
	if !strings.Contains(got, "proxy=http%3A%2F%2Fuser%3Apass%4045.56.171.66%3A7567") {
		t.Fatalf("Proxy.URL() missing current proxy: got=%q", got)
	}
}

func TestProxyURL_SpecialCharactersRoundTrip(t *testing.T) {
	t.Parallel()

	proxy := Proxy{
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     3128,
		Username: "first last@corp",
		Password: "p@ ss:#word",
	}

	parsed, err := url.Parse(proxy.URL())
	if err != nil {
		t.Fatalf("parse proxy URL failed: %v", err)
	}
	if got := parsed.User.Username(); got != proxy.Username {
		t.Fatalf("username mismatch after parse: got=%q want=%q", got, proxy.Username)
	}
	pass, ok := parsed.User.Password()
	if !ok {
		t.Fatal("password missing after parse")
	}
	if pass != proxy.Password {
		t.Fatalf("password mismatch after parse: got=%q want=%q", pass, proxy.Password)
	}
}
