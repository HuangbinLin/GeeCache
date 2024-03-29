package lfu

import (
	"fmt"
	"reflect"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	lfu := New(int64(0), nil)
	lfu.Add("key1", String("1234"))
	fmt.Println(lfu.maxBytes)
	if v, ok := lfu.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lfu.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lfu := New(int64(cap), nil)
	lfu.Add(k1, String(v1))
	lfu.Add(k2, String(v2))
	lfu.Add(k3, String(v3))
	fmt.Println(lfu)
	if _, ok := lfu.Get("key1"); ok || lfu.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lfu := New(int64(10), callback)
	lfu.Add("key1", String("123456"))
	lfu.Add("k2", String("k2"))
	lfu.Add("k2", String("k2"))
	lfu.Add("k3", String("k3"))
	lfu.Add("k4", String("k4"))
	lfu.Add("k5", String("k5"))
	lfu.Add("k5", String("k5"))

	expect := []string{"key1", "k3", "k4"}
	fmt.Println(keys)
	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}

func TestAdd(t *testing.T) {
	lfu := New(int64(0), nil)
	lfu.Add("key", String("1"))
	lfu.Add("key", String("111"))

	if lfu.nbytes != int64(len("key")+len("111")) {
		t.Fatal("expected 6 but got", lfu.nbytes)
	}
}
