GeeCache是一个模仿groupcache的分布式缓存实现，考虑了资源控制、淘汰策略、并发、分布式节点通信等各个方面的问题，基本功能如下
1. 结合LRU缓存策略和sync.Mutex实现服务器缓存的并发控制
2. 基于 HTTP 的分布式缓存和一致性哈希选择节点，实现负载均衡
3. 使用 Go 锁机制防止缓存击穿
4. 使用 protobuf 优化节点间二进制通信

更新：
1. 使用LFU缓存机制替代LRU
