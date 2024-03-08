package lfu

import (
	"container/list"
)

// Cache is a LFU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes   int64
	nbytes     int64
	minFreq    int
	cache      map[string]*list.Element
	freqToList map[int]*list.List
	OnEvicted  func(key string, value Value)
}

type entry struct {
	key   string
	value Value
	freq  int // Track the frequency of each entry
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:   maxBytes,
		nbytes:     0,
		freqToList: make(map[int]*list.List),
		cache:      make(map[string]*list.Element),
		OnEvicted:  onEvicted,
	}
}

func (c *Cache) pushFront(e *entry) {
	if _, ok := c.freqToList[e.freq]; !ok {
		c.freqToList[e.freq] = list.New() // 双向链表
	}
	c.cache[e.key] = c.freqToList[e.freq].PushFront(e)
}

func (c *Cache) getEntry(key string) *entry {
	node := c.cache[key]
	if node == nil { // 没有这本书
		return nil
	}
	e := node.Value.(*entry)
	lst := c.freqToList[e.freq]
	lst.Remove(node)    // 把这本书抽出来
	if lst.Len() == 0 { // 抽出来后，这摞书是空的
		delete(c.freqToList, e.freq) // 移除空链表
		if c.minFreq == e.freq {     // 这摞书是最左边的
			c.minFreq++
		}
	}
	e.freq++       // 看书次数 +1
	c.pushFront(e) // 放在右边这摞书的最上面
	return e
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if e := c.getEntry(key); e != nil { // 有这本书
		return e.value, true
	}
	return // 没有这本书
}

func (c *Cache) Add(key string, value Value) {
	if e := c.getEntry(key); e != nil { // 有这本书
		c.nbytes += int64(value.Len()) - int64(e.value.Len())
		e.value = value // 更新 value
		return
	} else {
		c.pushFront(&entry{key, value, 1}) // 新书放在「看过 1 次」的最上面
		c.minFreq = 1
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	for c.maxBytes != 0 && c.nbytes > c.maxBytes { // 书太多了
		lst := c.freqToList[c.minFreq] // 最左边那摞书
		if lst == nil {
			return
		}
		last_value := lst.Back().Value.(*entry)
		c.nbytes -= int64(len(last_value.key)) + int64(last_value.value.Len())

		delete(c.cache, lst.Remove(lst.Back()).(*entry).key) // 移除这摞书的最下面的书
		if lst.Len() == 0 {                                  // 这摞书是空的
			delete(c.freqToList, c.minFreq) // 移除空链表
		}
		c.updateMinFreq()
		if c.OnEvicted != nil {
			c.OnEvicted(last_value.key, last_value.value)
		}
	}
}

func (c *Cache) Len() int {
	length := 0
	for _, Element := range c.freqToList {
		length += Element.Len()
	}
	return length
}

func (c *Cache) updateMinFreq() {
	minFreq := -1
	for freq := range c.freqToList {
		if minFreq == -1 || (freq < minFreq && c.freqToList[freq].Len() > 0) {
			minFreq = freq
		}
	}
	c.minFreq = minFreq
}
