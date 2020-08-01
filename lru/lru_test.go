package lru
import "testing"
import "reflect"
type String string

func (d String) Len() int64{
	return int64(len(d))
}


func TestGet(t *testing.T){

	lru := New(int64(0), nil)
	lru.Add("key1", String("123"))
	if v,ok:= lru.Get("key1"); !ok|| string(v.(String))!="123" {
		t.Fatalf("cache hit key1 failed!")
	}

	if _,ok:= lru.Get("key2"); ok{
		t.Fatalf("cache miss key1 failed!")
	}

}


func TestRemoveoldest(t *testing.T){
	k1,k2,k3 := "k1","k2","k3"
	v1,v2,v3 := "va1","va2","va3"
	cap:= len(k1+v1)
	//t.Logf("%d maxbytes",cap)

	lru:= New(int64(cap),nil)
	t.Logf("%d maxbytes",lru.maxBytes)

	lru.Add(k1,String(v1))
	t.Logf("1. %d nbytes %d eles",lru.nbytes,lru.Len())

	lru.Add(k2,String(v2))

	t.Logf("2. %d nbytes %d eles",lru.nbytes,lru.Len())
	lru.Add(k3,String(v3))

	t.Logf("3. %d nbytes %d eles",lru.nbytes,lru.Len())

	if _,ok:= lru.Get(k1); ok || lru.Len() !=1 {
		t.Fatalf("Removeoldest k1 failed")
	}

	if _,ok:= lru.Get(k2); ok || lru.Len() !=1 {
		t.Fatalf("Removeoldest k1 failed")
	}

}


func TestOnEvicted(t *testing.T){
	keys := make([]string,0)
	cb := func(key string, value Value){
		keys = append(keys,key)
	}

	lru:= New(int64(10),cb)
	lru.Add("key1", String("1234"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))
	
	expect := []string{"key1","k2"}
	
	t.Logf("%v",keys)

	if !reflect.DeepEqual(expect, keys){
		t.Fatalf("Call onEvicted failed!, %s", expect)
	}
}