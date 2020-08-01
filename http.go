package gocache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath ="/_gocache"

type HTTPPOOL struct{
	self string
	basePath string
}

func NewHTTPPool(self string) *HTTPPOOL{
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
		panic("HTTPPOOL serving unexpected path: ",+ r.URL.Path)
	}

}