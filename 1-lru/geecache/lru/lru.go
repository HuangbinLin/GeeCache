package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
// 通过Cache结构体维护LRU，有New、Get、Add、RemoveOldest、Len方法
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List               // 这个Cache维护一个双向链表
	cache    map[string]*list.Element // 可以通过key直接找到对应的链表节点
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface { // 接口包含动态类型和动态值，动态类型指的是被接口值包含的实际类型，而动态值则是被接口值包含的实际值。
	Len() int // 限制输入的数据需要带有Len函数
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),                     // 返回一个双向链表
		cache:     make(map[string]*list.Element), // 根据key指向双向两边中的节点
		OnEvicted: onEvicted,
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { // 如果在里面，则把他移动到前面来
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)                               // 通过ele.Value.(*entry) 将链表元素的值转换为 *entry 类型的指针
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // 键一样，更新值的长度
		kv.value = value
	} else {
		// 创建一个新节点，推送到前面去
		// 这个节点的Value是any，需要满足的是包含Len函数
		ele := c.ll.PushFront(&entry{key, value})
		// 赋值了&entry{key, value}
		c.cache[key] = ele                               // key指向节点
		c.nbytes += int64(len(key)) + int64(value.Len()) // 这个缓存保存了多少数据
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest() // 移除最旧的
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 返回最后一个
	if ele != nil {
		c.ll.Remove(ele) // 移除节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                // 删除map中的对应key和值
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 返回大小
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
