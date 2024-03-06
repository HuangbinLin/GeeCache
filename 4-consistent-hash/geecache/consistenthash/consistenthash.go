package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// 提供的哈希一致性环的计算过程Add，还有获取查询key所对应节点的Get
// Map constains all hashed keys
type Map struct {
	hash     Hash           // hash计算函数，可以自定义
	replicas int            // 虚拟节点的个数，直接在前面加(0,1,...,replicas - 1)
	keys     []int          // Sorted  // 环
	hashMap  map[int]string // 节点映射回key
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 将整数i转换为字符串 + key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash) // hash值是整数
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys) // 排序
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	// 返回找到的第一个满足条件的索引值，找不到则返回len(m.keys)
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 如果idx == len(m.keys)，则把他定位给第0个节点
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
