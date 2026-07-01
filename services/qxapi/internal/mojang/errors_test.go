package mojang

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestClassifyAuthError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   error
		want error
	}{
		{
			name: "invalid grant",
			in:   fmt.Errorf("ms token: http 400: invalid_grant"),
			want: ErrSessionRevoked,
		},
		{
			name: "token unauthorized",
			in:   fmt.Errorf("ms token: http 401: unauthorized"),
			want: ErrSessionRevoked,
		},
		{
			name: "invalid client",
			in:   fmt.Errorf("ms token: http 401: invalid_client"),
			want: ErrNotConfigured,
		},
		{
			name: "upstream 503",
			in:   fmt.Errorf("xbl auth: http 503: busy"),
			want: ErrSessionUnavailable,
		},
		{
			name: "network timeout",
			in:   &url.Error{Op: "Post", URL: "https://login.live.com", Err: &net.DNSError{IsTimeout: true}},
			want: ErrSessionUnavailable,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyAuthError(tc.in)
			if !errors.Is(got, tc.want) {
				t.Fatalf("ClassifyAuthError(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
