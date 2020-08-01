package lru

import "container/list"
//import "fmt"

type Cache struct{
	maxBytes int64 // Allow max use memo
	nbytes int64  // current used memo
	ll *list.List
	cache map[string]*list.Element

	OnEvicted func (key string, value Value) 
}

// entry is the element of list
type entry struct{
	key string
	value Value
}

// Value type : general value
// Len calc used bytes
type Value interface{
	Len() int64
}

func New(maxBytes int64, onEvicted func(string,Value)) *Cache{
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool){
	if ele,ok := c.cache[key]; ok{
		c.ll.MoveToFront(ele)
		kv:= ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *Cache) RemoveOldest(){
	ele:=c.ll.Back()
	if ele!=nil{
		
		c.ll.Remove(ele)
		kv:= ele.Value.(*entry)
		delete(c.cache, kv.key)
		//fmt.Printf("delet %s\n",kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil{
			c.OnEvicted(kv.key,kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value){
	if ele,ok := c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv:= ele.Value.(*entry)
		c.nbytes+= int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	}else{
		ele:= c.ll.PushFront(&entry{key,value})
		c.cache[key] = ele
		c.nbytes+= int64(len(key)) + int64(value.Len())
	}

	for c.maxBytes!=0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}

}

func (c *Cache) Len() int{
	return c.ll.Len()
}