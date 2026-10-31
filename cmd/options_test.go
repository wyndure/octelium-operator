package main

import (
	"errors"
	"flag"
	"io"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		interval time.Duration
		invalid  bool
	}{
		{name: "defaults", interval: 30 * time.Second},
		{name: "custom interval", args: []string{"--sync-interval=2m", "--leader-elect", "--metrics-bind-address=:9090"}, interval: 2 * time.Minute},
		{name: "zero interval", args: []string{"--sync-interval=0s"}, invalid: true},
		{name: "negative interval", args: []string{"--sync-interval=-1s"}, invalid: true},
		{name: "invalid interval", args: []string{"--sync-interval=tomorrow"}, invalid: true},
		{name: "positional arguments", args: []string{"unexpected"}, invalid: true},
		{name: "unknown flag", args: []string{"--unknown"}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			options, err := parseOptions(test.args, io.Discard)
			if test.invalid {
				if err == nil {
					t.Fatal("invalid arguments accepted")
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if options.syncInterval != test.interval {
				t.Fatalf("interval = %v, want %v", options.syncInterval, test.interval)
			}
		})
	}
}

func TestHelp(t *testing.T) {
	_, err := parseOptions([]string{"--help"}, io.Discard)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("help returned %v", err)
	}
}
