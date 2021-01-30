package ecs_instance_replace_test

import (
	"reflect"
	"testing"

	"github.com/yasuoza/ecs_instance_replace"
)

func TestSliceDifference(t *testing.T) {
	tests := []struct {
		a    []string
		b    []string
		want []string
	}{
		{
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "d"},
			want: []string{"c"},
		},
		{
			a:    []string{"a", "b", "c", "d"},
			b:    []string{"a", "b", "d"},
			want: []string{"c"},
		},
		{
			a:    []string{"a", "b", "c"},
			b:    []string{},
			want: []string{"a", "b", "c"},
		},
		{
			a:    []string{"a", "b", "c"},
			b:    []string{"d", "e", "f"},
			want: []string{"a", "b", "c"},
		},
		{
			a:    []string{},
			b:    []string{"a", "b", "c"},
			want: nil, // []
		},
		{
			a:    []string{},
			b:    []string{},
			want: nil, // []
		},
	}

	for _, test := range tests {
		got := ecs_instance_replace.SliceDifference(test.a, test.b)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("want %v, but got %v\n", test.want, got)
		}
	}
}
