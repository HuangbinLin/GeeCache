package geecache

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
	"log"
	"sync"
)

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string     // 缓存命名空间
	mainCache cache      // 带锁的LRU缓存
	getter    Getter     // 节点到value的选择策略
	peers     PeerPicker // key到节点的选择策略
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group // 防止穿透
}

// A Getter loads data for a key.
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (interface{}, error) { // DO带有防止穿透
		if g.peers != nil {
			// 输入key，返回一个选择的节点，这个节点带有Get功能
			// g.peers = PeerPicker,PeerPicker是一个接口
			// PickPeer是一个接口里面的函数，具体的是怎么实现看g.peers
			// g.peers是peers赋值的，peers是NewHTTPPool这个结构体实现
			// NewHTTPPool带有hash环，和分布式节点映射
			// NewHTTPPool的PickPeer返回一个节点映射后的地址结构体
			// 映射后的地址结构体地址为http://localhost:800X" + "/_geecache/
			// 地址结构体实现了Get函数
			if peer, ok := g.peers.PickPeer(key); ok {
				// peer是节点映射后的地址结构体，他实现了GET函数，作为PeerGetter接口
				// getFromPeer调用了Get，输出整理后的数据
				// 由Protobuf帮忙向子节点发起通信
				// h.baseURL,//url.QueryEscape(in.GetGroup()),//url.QueryEscape(in.GetKey()),
				// 网址为 baseURL + http://localhost:800X" + "/_geecache/ + Group + key
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
