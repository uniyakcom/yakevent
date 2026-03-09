// Package pool 提供高性能事件对象池 + 自动 Arena 管理
//
// 设计：
//   - Acquire/Release 管理 Event 对象复用
//   - EnableArena=true 时，AllocData 自动从 Arena 分配（无 malloc）
//   - Arena 满时自动切换新 chunk，对调用侧透明
package pool

import (
	"sync"
	"sync/atomic"

	"github.com/uniyakcom/yakevent/core"
	"github.com/uniyakcom/yakevent/internal/support/noop"
)

const arenaChunkSize = 64 * 1024

type ArenaChunk struct {
	buf    []byte
	offset atomic.Int64
}

// Alloc CAS bump allocator — 无锁热路径
func (a *ArenaChunk) Alloc(n int) []byte {
	aligned := int64((n + 7) &^ 7)
	for {
		cur := a.offset.Load()
		next := cur + aligned
		if next > int64(len(a.buf)) {
			return nil
		}
		if a.offset.CompareAndSwap(cur, next) {
			return a.buf[cur : cur+int64(n) : cur+aligned]
		}
	}
}

// ─── Event Pool ──────────────────────────────────────────────────────

// EventPool 高性能事件对象池 + 自动 Arena 管理
type EventPool struct {
	pool         sync.Pool
	enableArena  atomic.Bool
	currentArena *ArenaChunk
	arenaLock    noop.Mutex
	chunkPool    sync.Pool
}

// New 创建事件对象池
func New() *EventPool {
	p := &EventPool{
		pool: sync.Pool{
			New: func() interface{} { return &core.Event{} },
		},
		arenaLock: noop.NewMutex(true),
		chunkPool: sync.Pool{
			New: func() interface{} {
				return &ArenaChunk{buf: make([]byte, arenaChunkSize)}
			},
		},
	}
	p.enableArena.Store(true)
	p.currentArena = p.chunkPool.Get().(*ArenaChunk)
	return p
}

// Acquire 获取 Event（零分配热路径）
func (p *EventPool) Acquire() *core.Event {
	return p.pool.Get().(*core.Event)
}

// Release 归还 Event（最小化清零后放回池）
func (p *EventPool) Release(evt *core.Event) {
	if evt == nil {
		return
	}
	evt.Data = nil
	evt.Type = ""
	p.pool.Put(evt)
}

// AllocData 分配 Data
func (p *EventPool) AllocData(n int) []byte {
	if !p.enableArena.Load() {
		return make([]byte, n)
	}
	if n > arenaChunkSize/2 {
		return make([]byte, n)
	}
	arena := p.currentArena
	if buf := arena.Alloc(n); buf != nil {
		return buf
	}
	p.arenaLock.Lock()
	arena = p.currentArena
	if buf := arena.Alloc(n); buf != nil {
		p.arenaLock.Unlock()
		return buf
	}
	newChunk := p.chunkPool.Get().(*ArenaChunk)
	newChunk.offset.Store(0)
	oldChunk := p.currentArena
	p.currentArena = newChunk
	buf := newChunk.Alloc(n)
	p.arenaLock.Unlock()
	p.chunkPool.Put(oldChunk)
	return buf
}

// ─── 全局便捷接口 ───────────────────────────────────────────────────

var global = New()

// Acquire 全局获取 Event
func Acquire() *core.Event { return global.Acquire() }

// Release 全局归还 Event
func Release(evt *core.Event) { global.Release(evt) }

// AllocData 全局分配 Data
func AllocData(n int) []byte { return global.AllocData(n) }

// Global 获取全局池引用
func Global() *EventPool { return global }

// SetEnableArena 全局设置是否启用 Arena
func SetEnableArena(enable bool) { global.enableArena.Store(enable) }
