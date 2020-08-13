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

// cacheserver其实就是 HTTPPOOL， 更上层一点
func startCacheServer(addr string, addrs []string, gee *gocache.Group){
	// 这句其实 变量的命名不太合适，首先peers是 HTTPPOOL的一个成员，本身是consis.map
	// 直接用 peers来表示 HTTPPOOL的实例 有点扩大化 peers的语义了
	// 好像原作者没觉得有什么问题？？
	peers := gocache.NewHTTPPOOL(addr)

	// 开节点
	peers.Set(addrs...)
	// 存储 对接节点
	gee.RegisterPeers(peers)

	log.Println("gocache is running at" ,addr)

	// 这个其实也可以查了，调用的是 ServeHTTP
	log.Fatal(http.ListenAndServe(addr[7:],peers))
	// 但开了下面的东西转接了一下
}

func startAPIServer(apiAddr string, gee *gocache.Group){
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request){
			key := r.URL.Query().Get("key")
			
			// 这里的查是 gee.Get 和 上面listenAnd()里开的peers其实有同样的功能，
			// 这里反而有点破坏了原有封装的感觉
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

	// run.sh 开了三次 所以gee每次都是新的
	// 所以每次注册 peers的都是新的，最后的peers是 8003
	// 解释了为什么第一个log 显示的是 [server 8003]peerpick , 因为gee持有的 peers 是8003，调用了  peerpick，而peerpick内打了log，log会带原server的属性
	// 第二个log 是 [8001] 比较易见，因为key映射去了 8001，回去 然后调用http.Get，被自己serverHTTP检测到了
	gee := createGroup()
	if api{
		go startAPIServer(apiAddr,gee)
	}
	
	startCacheServer(addrMap[port],[]string(addrs),gee)
}