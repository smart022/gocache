package gocache
import (
	"fmt"
	"sync"
	"log"
	"./singleflight"
	pb "./gocachepb"
)

type Getter interface{
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)


func (f GetterFunc) Get(key string) ([]byte, error){
	return f(key)
}


type Group struct {
	name string
	getter Getter
	mainCache cache
	// 5th
	peers PeerPicker
	// 6th
	loader *singleflight.Group
}


// 歪日，这个groups竟然是个全局的map。。。
var (
	mu sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup竟然是在这个全局的 map上注册。。。，我就说最后的httppool都没有持有 group，原来要用的时候直接全局查map就好了。。。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group{
	if getter == nil{
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{cacheBytes:cacheBytes},
		loader: &singleflight.Group{},
	}

	groups[name] = g
	return g
}

func GetGroup(name string) *Group{
	mu.RLock()
	g:=groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error){
	if key==""{
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v,ok:= g.mainCache.get(key);ok{
		log.Println("[Gocache] hit")
		return v,nil
	}

	// not in cache 
	return g.load(key)
}

// peers 是另外注册的，意思注册了的和没有注册的是两种
// 注册了就可以 PickPeer了
func (g *Group) RegisterPeers(peers PeerPicker){ 
	if g.peers != nil{
		panic("RegisterPeerPicker called more than once!")
	}
	g.peers = peers
}

// 本地无缓存了，往外找
func (g *Group) load(key string) (value ByteView, err error){

	// 这样保证每个key 只被拉一次 无论远程或者local
	// 无论并发数量
	viewi, err:= g.loader.Do(key, func() (interface{},error) {
		
		// 原代码内置
		if g.peers != nil{
			if peer,ok := g.peers.PickPeer(key); ok{
				if value,err = g.getFromPeer(peer,key); err == nil{
					return value,nil
				}
				log.Println("[Gocache] Failed to get from peer",err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil{
		return viewi.(ByteView), nil
	}

	return 
}

func (g *Group) getFromPeer(peer PeerGetter,key string) (ByteView, error){
	req := &pb.Request{
		Group: g.name,
		Key: key,
	}
	res := &pb.Response{}
	
	err :=peer.Get(req,res)
	if err!=nil{
		return ByteView{},err
	}
	return ByteView{b: res.Value},nil
}

func (g *Group) getLocally(key string) (ByteView, error){
	bytes,err := g.getter.Get(key)
	if err != nil{
		return ByteView{},err
	}

	value := ByteView{b: cloneBytes(bytes)}
	// add in cache
	g.populateCache(key,value)
	return value , nil
}

func (g *Group) populateCache(key string, value ByteView){
	g.mainCache.add(key,value)
}