package singleflight

import (
	"testing"
)

func TestDo(t *testing.T) {
	var g Group
	v, err := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})

	for i := 0; i < 100; i++ {
		go g.Do("key", func() (interface{}, error) {
			return "bar", nil
		})
	}

	if v != "bar" || err != nil {
		t.Errorf("Do v = %v, error = %v", v, err)
	}
}
