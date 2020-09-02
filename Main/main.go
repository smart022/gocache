package main

// main放置于gocache包上层
import (
	"log"
	"./gocache"
	"net/http"
	"fmt"
	"flag"
)

var db = map[string]string{
	"Tim":"638",
	"Jack":"568",
	"SS":"666",
}

// group是对接存储的，更下一点；
// 工作内容查，1本地 2远程 3回调
func createGroup() *gocache.Group{
	return gocache.NewGroup("scores",2<<10, gocache.GetterFunc(
		func (key string) ([]byte,error) {
			log.Println("[SlowDB] search key", key)
			if v,ok := db[key];ok{
				return []byte(v),nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// HTTPPOOL，是最核心的结构，代表了一个分布式节点-相互通信
// --------------------
// cacheserver其实就是 HTTPPOOL， 更上层一点。 cache http 理应不暴露，用户透明
func startCacheServer(addr string, addrs []string, gee *gocache.Group){
	// 这句其实 变量的命名不太合适，首先peers是 HTTPPOOL的一个成员，本身是consis.map
	// 直接用 peers来表示 HTTPPOOL的实例 有点扩大化 peers的语义了
	// 好像原作者没觉得有什么问题？？
	// -------------- 上述是我之前理解不够
	// 其实peers首先是个抽象类， httppool是实现
	// 至于内部有个peers的成员，也只是用来管理这个类似的概念，内部成员是个狭义的peers，（且具体的实现 一致哈希）
	// 用来挑选key的owener的
	peers := gocache.NewHTTPPOOL(addr)

	// 注册环路映射
	peers.Set(addrs...)

	// 存储 对接 http 节点
	gee.RegisterPeers(peers)


	// 本质上一个 http节点会与 一个 gee group 对应，
	// 但 http 节点会有个 分布式环路映射， gee grooup 也可注册这个， 使得其可查询 对分布式环路

	log.Println("gocache is running at" ,addr)

	// 这个其实也可以查了，调用的是 ServeHTTP
	log.Fatal(http.ListenAndServe(addr[7:],peers))
	// 但开了下面的东西转接了一下
}

// API 是暴露给用户的，直接使用
// 这个api 很简单， 就是裹了一层的http 暴露这个查询api node，忽略真实的cache http node，替你查gee group
func startAPIServer(apiAddr string, gee *gocache.Group){
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request){
			key := r.URL.Query().Get("key")
			
			// 这里的查是 gee.Get 和 上面listenAnd()里开的peers其实有同样的功能，
			// 这里反而有点破坏了原有封装的感觉 ps: 没问题就是 用gee group 来查的

			// 没走 cache http的查groupname 定位gee的
			// 直接gee 绑定了。。。
			// 但 gee 又反向注册了 分布式环路，在本地没有缓存的时候，走分布式，如果这个key的映射是自己的话 就会产生缓存了。
			view,err :=  gee.Get(key)
			if err!= nil{
				http.Error(w,err.Error(),http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type","application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:],nil))
}

func main(){
	var port int
	var api bool

	flag.IntVar(&port,"port",8001,"Gocache server port")
	flag.BoolVar(&api,"api",false,"Start an api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap:= map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _,v := range addrMap{
		addrs = append(addrs,v)
	}

	// 要区分一下，通信节点 和 通信api 的区别： 节点都有gee但不支持直接用户访问，除非这个节点另外开了个端口给用户通信
	// 给用户使用的意思是，直接走curl可以调用gee.get()，

	// run.sh 开了三次 所以gee每次都是新的（严格的说，每个gee的名称都一样，开了三个进程而已） (old)
	// createGroup 调用了 NewAGroup , 里面用全局map来标定一个 group (old)
	// 上述不太对的地方：每个节点 肯定要带一个gee，这是基础结构啊，肯定每个节点都有一个
	
	// 所以每次startCacheServer注册peers的都是新的 替换掉的，最后的peers是 8003 (old)
	// peers肯定每个节点

	// 解释了为什么第一个log 显示的是 [server 8003]peerpick , 因为gee持有的 peers 是8003，调用了  peerpick，而peerpick内打了log，log会带原server的属性
	// 第二个log 是 [8001] 比较易见，因为key映射去了 8001，回去 然后调用http.Get，被自己serverHTTP检测到了
	gee := createGroup()
	if api{
		go startAPIServer(apiAddr,gee)
	}
	
	startCacheServer(addrMap[port],[]string(addrs),gee)
}

/*
结果展示
>>> start test
2020/08/14 10:47:14 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/08/14 10:47:14 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/08/14 10:47:14 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/08/14 10:47:14 [Server http://localhost:8001] GET /_gocache/scores/Tim
2020/08/14 10:47:14 [SlowDB] search key Tim
2020/08/14 10:47:14 [Server http://localhost:8001] GET /_gocache/scores/Tim
2020/08/14 10:47:14 [Gocache] hit
2020/08/14 10:47:14 [Server http://localhost:8001] GET /_gocache/scores/Tim
2020/08/14 10:47:14 [Gocache] hit


*/