package algo

import (
	"cmp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_quickSort(t *testing.T) {
	type args[T comparable] struct {
		s   []T
		cmp func(a, b T) int
	}
	type testCase[T comparable] struct {
		name string
		args args[T]
	}
	tests := []testCase[string]{
		{
			"one", args[string]{
				[]string{"a", "c", "m", "a"},
				func(a, b string) int {
					return cmp.Compare(a, b)
				},
			},
		},
		{
			"two", args[string]{
				[]string{"abcd", "m", "m", "a", "def", "777", "123", "&&&"},
				func(a, b string) int {
					return cmp.Compare(a, b)
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quickSort(tt.args.s, tt.args.cmp)
			r := append([]string(nil), tt.args.s...)
			sort.Strings(r)
			assert.Equal(t, r, tt.args.s)
		})
	}
}

// func sortBySort[]([]) {}
