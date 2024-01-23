package main

import (
	"testing"
	"time"

	"github.com/mabego/snippetbox-mysql/internal/assert"
)

func TestHumanDate(t *testing.T) {
	// A slice of anonymous structs containing the test data.
	tests := []struct {
		name string
		tm   time.Time
		want string
	}{
		{
			name: "UTC",
			tm:   time.Date(2023, 7, 19, 10, 15, 0, 0, time.UTC),
			want: "Jul 19 2023 at 10:15",
		},
		{
			name: "Empty",
			tm:   time.Time{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hd := humanDate(tt.tm)
			assert.Equal(t, hd, tt.want)
		})
	}
}
