package singleflight
/*
	解决缓存雪崩、击穿、穿透等问题

	对同一个key 不需要每次都访问映射的节点
*/
import "sync"

// 实现看一看这里 https://zhuanlan.zhihu.com/p/75441551
// 其实照搬了 groupcache 的实现

/*
	
因为这个singleflight机制相当于是一个请求的缓冲器，不需要有储存功能。

	在少量访问时，正常使用。
	在大量并发访问时，对于并发的信息，共享第一个请求的返回值，大幅减少请求次数

*/

type call struct{
	wg sync.WaitGroup
	val interface{}
	err error
}


// 又是Group 同名概念很容易混淆
type Group struct{
	mu sync.Mutex
	m map[string]*call
}

// 如何思考这个用法，想象两个同时执行的Do的协程
func (g *Group) Do(key string, fn func() (interface{},error)) (interface{},error){
	g.mu.Lock()
	if g.m == nil{
		g.m = make(map[string]*call)
	}

	// g.mu 保护 g.m[]的读写
	if c,ok:= g.m[key];ok{
		g.mu.Unlock()
		c.wg.Wait()
		return c.val ,c.err
	}

	c:= new(call)
	c.wg.Add(1) // 请求前加锁
	g.m[key] = c // 添加到 g.m 表明key 已有请求在处理

	// g.mu 保护 g.m[]的读写
	g.mu.Unlock()

	c.val, c.err = fn() // 调用 fn, 发起请求
	c.wg.Done() // 请求结束

	// 为什么会delete掉，因为这singleflight 就只是个防止瞬间被击穿的功能，不持久保存
	g.mu.Lock()
	delete(g.m,key) // 更新 g.m
	g.mu.Unlock()

	return c.val, c.err

}

