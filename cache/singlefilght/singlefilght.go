package singlefilght

import (
	"sync"
)

// 代表进行中或已结束的请求，wg避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 是singleFlight主体结构，管理不同key的call请求
// singleFlight防止短时间对统一资源的重复请求（去重）
type Group struct {
	mu sync.Mutex // protects m
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 存在相同key的请求正在进行或刚进行完
	if cal, ok := g.m[key]; ok {
		g.mu.Unlock()
		cal.wg.Wait() // 等待正在进行的请求结束
		return cal.val, cal.err
	}

	// key没有正在进行请求，新建一个请求
	cal := new(call)
	cal.wg.Add(1)
	g.m[key] = cal
	g.mu.Unlock()

	// 处理该key
	cal.val, cal.err = fn()
	cal.wg.Done() // 设置该请求完成

	// 删除该key，因为只是防止短时间高并发，同时数据可能会变化，不能一直存着旧值
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, key)

	return cal.val, cal.err
}
