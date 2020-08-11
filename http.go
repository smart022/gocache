package gocache

import (
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"./consistenthash"
)

const (
	defaultBasePath ="/_gocache"
	defaultRepicas = 50
)
// httppool implements PeerPicker for a pool of HTTP peers
type HTTPPOOL struct{
	self string
	basePath string
	mu sync.Mutex // guards peers and httpGetters
	peers *consistenthash.Map
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

func (p *HTTPPOOL) ServeHTTP(w http.ResponseWriter, r *http.Request){
	if !strings.HasPrefix(r.URL.Path, p.basePath){
		panic("HTTPPOOL serving unexpected path: "+ r.URL.Path)
	}

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

	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(view.ByteSlice())

}


// ----------- client

type httpGetter struct{
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error){
	u:= fmt.Sprintf(
		"%v%v%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err!=nil{
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK{
		return nil, fmt.Errorf("server returned: %v",res.Status)
	}

	bytes, err:= ioutil.ReadAll(res.Body)
	if err!=nil{
		return nil, fmt.Errorf("reading response body: %v",err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

