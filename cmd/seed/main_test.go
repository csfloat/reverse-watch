package main

import "testing"

func TestIsLocalHost(t *testing.T) {
	testCases := []struct {
		host string
		want bool
	}{
		{host: "", want: true},
		{host: "localhost", want: true},
		{host: "127.0.0.1", want: true},
		{host: "::1", want: true},
		{host: "/var/run/postgresql", want: true},
		{host: "postgres", want: true},
		{host: "db", want: true},
		{host: "10.0.0.5", want: false},
		{host: "db.example.com", want: false},
		{host: "my-project:us-central1:reverse-watch", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.host, func(t *testing.T) {
			if got := isLocalHost(tc.host); got != tc.want {
				t.Errorf("isLocalHost(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}
