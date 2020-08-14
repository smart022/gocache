package gocache

import (
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"./consistenthash"
	pb "./gocachepb"
	"github.com/golang/protobuf/proto"
)

const (
	defaultBasePath ="/_gocache"
	defaultRepicas = 50
)
// httppool implements PeerPicker for a pool of HTTP peers
// 所谓池，其实就是hold住了一圈http节点
// 并不是类似线程池的概念，就是容器
type HTTPPOOL struct{
	self string // 自己的 ip
	basePath string
	// 5th new adding
	mu sync.Mutex // guards peers and httpGetters
	peers *consistenthash.Map // peers 就是一致性哈希的那个圈 (ps: 节点string 一般是ip
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.1:9999"
}

func NewHTTPPOOL(self string) *HTTPPOOL{
	return &HTTPPOOL{
		self: self,
		basePath: defaultBasePath,
	}
}


func (p *HTTPPOOL) Log(format string, v ... interface{}){
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 这是原始的靠分辨 GET /groupname/key 来查
func (p *HTTPPOOL) ServeHTTP(w http.ResponseWriter, r *http.Request){
	if !strings.HasPrefix(r.URL.Path, p.basePath){
		panic("HTTPPOOL serving unexpected path: "+ r.URL.Path)
	}

	// 这里收到了 转移的GET 请求
	p.Log("%s %s",r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):],"/",3)
	//log.Println("0 Debug: ",r.URL.Path[len(p.basePath):]," parts: ",parts)
	if len(parts)!=3{
		http.Error(w,"bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[1]
	key := parts[2]

	group := GetGroup(groupName)

	if group == nil{
		//log.Println("1 Debug: ",groupName ," key: ",key)
		http.Error(w,"no such group: "+groupName,http.StatusNotFound)
		return
	}

	view,err := group.Get(key)
	if err !=nil{
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	//adding proto , data in proto format
	body,err:= proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err!=nil{
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(body)

}

// 5th adding
// Set updates the pool's list of peers
func (p* HTTPPOOL) Set(peers ...string){
	p.mu.Lock()
	defer p.mu.Unlock()

	// 狭义 peers consis.Map
	p.peers = consistenthash.New(defaultRepicas,nil)
	p.peers.Add(peers...)

	// 环路节点 名字再绑定个 handle
	p.httpGetters = make(map[string]*httpGetter,len(peers))
	for _,peer:= range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer+p.basePath}
	}
}


func (p *HTTPPOOL) PickPeer(key string) (PeerGetter, bool){
	p.mu.Lock()
	defer p.mu.Unlock()

	// 狭义 consis.Map 查询key落入对应的 ip
	if peer:= p.peers.Get(key); peer!="" && peer!=p.self{
		p.Log("Pick peer %s",peer)
		return p.httpGetters[peer], true
	}
	return nil,false
}



// ----------- client

type httpGetter struct{
	baseURL string
}

//func (h *httpGetter) Get(group string, key string) ([]byte, error){
func (h *httpGetter) Get(in *pb.Request, out *pb.Response)  error {	
	u:= fmt.Sprintf(
		"%v/%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()), // group
		url.QueryEscape(in.GetKey()), // key
	)
	res, err := http.Get(u)
	if err!=nil{
		return  err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK{
		// nil, fmt.Errorf("server returned: %v",res.Status)
		return  fmt.Errorf("server returned: %v",res.Status)
	}

	bytes, err:= ioutil.ReadAll(res.Body)
	if err!=nil{
		return  fmt.Errorf("reading response body: %v",err)
	}

	//
	if err = proto.Unmarshal(bytes,out);err!=nil{
		return fmt.Errorf("decoding response body: %v",err)
	}

	// bytes,nil
	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

