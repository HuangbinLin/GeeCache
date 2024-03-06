package main

/*
$ curl "http://localhost:9999/api?key=Tom"
630

$ curl "http://localhost:9999/api?key=kkk"
kkk not exist
*/

// 转变中间载体
import (
	"flag"
	"fmt"
	"geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr) // 注册self为addr(服务器端)，basePath为"/_geecache/"
	peers.Set(addrs...)                 // 注册hash一致性的Replicas，addrs作为key进行hash环注册
	gee.RegisterPeers(peers)            // 注册peers放入Group
	log.Println("geecache is running at", addr)
	// peers需要实现ServeHTTP
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil)) // 指定了监听地址

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port") // 命令行 -port参数输入
	flag.BoolVar(&api, "api", false, "Start a api server?")  // 命令行 -api参数输入
	flag.Parse()                                             // flag.Parse() 函数来解析命令行参数。这个函数将扫描命令行参数。

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup() // score的缓存空间
	// 如果有API则使用协程启动"localhost:9999/api"的监听端口
	if api { //注册所有以"/api"开头的HTTP请求的处理器。可以理解为注册了"api"组
		// 启动"localhost:9999/api"的监听端口，从命令行解析到key，执行gee.Get(key)
		go startAPIServer(apiAddr, gee)
	}
	// 没有API则启动"localhost:9999/_geecache/"端口
	// 在Gruop中注册了服务器地址，hash环的创建。 查询key->查询hash->虚拟节点->节点->节点地址
	startCacheServer(addrMap[port], addrs, gee)
}
