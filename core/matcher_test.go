package core

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

// TestTrieMatcherExact 验证精确匹配快速路径：精确 pattern 命中 exact sync.Map。
func TestTrieMatcherExact(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("user.created")
	m.Add("order.paid")

	tests := []struct {
		event string
		want  []string
	}{
		{"user.created", []string{"user.created"}},
		{"order.paid", []string{"order.paid"}},
		{"user.deleted", nil},
		{"order", nil},
	}

	for _, tc := range tests {
		sp := m.Match(tc.event)
		got := append([]string{}, *sp...)
		m.Put(sp)
		sort.Strings(got)
		sort.Strings(tc.want)
		if !strSliceEqual(got, tc.want) {
			t.Errorf("Match(%q) = %v, want %v", tc.event, got, tc.want)
		}
	}
}

// TestTrieMatcherExactFastPath 验证精确 pattern 存在时走 exact 快速路径（不走 trie）。
// 设计说明：当事件类型本身被注册为精确 pattern，exact 快速路径直接返回，
// 不合并通配符匹配结果。此行为是有意为之的性能选择。
func TestTrieMatcherExactFastPath(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("user.created") // exact pattern

	// 精确快速路径只返回精确 pattern
	sp := m.Match("user.created")
	count := len(*sp)
	m.Put(sp)
	if count != 1 {
		t.Errorf("exact fast path: count=%d, want 1", count)
	}
}

// TestTrieMatcherSingleWildcard 验证 * 单层通配符。
func TestTrieMatcherSingleWildcard(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("user.*")

	tests := []struct {
		event string
		match bool
	}{
		{"user.created", true},
		{"user.deleted", true},
		{"user.profile.updated", false}, // * 不跨层
		{"order.created", false},
		{"user", false},
	}
	for _, tc := range tests {
		sp := m.Match(tc.event)
		got := len(*sp) > 0
		m.Put(sp)
		if got != tc.match {
			t.Errorf("Match(%q): got match=%v, want %v", tc.event, got, tc.match)
		}
	}
}

// TestTrieMatcherMultiWildcard 验证 ** 多层通配符。
func TestTrieMatcherMultiWildcard(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("user.**")

	tests := []struct {
		event string
		match bool
	}{
		{"user.created", true},
		{"user.profile.updated", true},
		{"user.a.b.c", true},
		{"order.created", false},
	}
	for _, tc := range tests {
		sp := m.Match(tc.event)
		got := len(*sp) > 0
		m.Put(sp)
		if got != tc.match {
			t.Errorf("Match(%q) match=%v, want %v", tc.event, got, tc.match)
		}
	}
}

// TestTrieMatcherMixed 两个通配符 pattern 同时匹配同一事件。
// 注意：精确 pattern 不加入，避免触发 exact 快速路径，确保 trie 路径被测试。
func TestTrieMatcherMixed(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("user.*")
	m.Add("user.**")

	// 仅通配符时走 trie 慢路径，两者均应匹配 "user.created"
	sp := m.Match("user.created")
	got := append([]string{}, *sp...)
	m.Put(sp)
	sort.Strings(got)

	want := []string{"user.*", "user.**"}
	sort.Strings(want)

	if !strSliceEqual(got, want) {
		t.Errorf("mixed wildcard Match = %v, want %v", got, want)
	}
}

// TestTrieMatcherHasMatch 验证 HasMatch 零分配快速路径。
func TestTrieMatcherHasMatch(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("order.*")
	m.Add("payment.processed")

	if !m.HasMatch("order.created") {
		t.Error("HasMatch(order.created) should be true")
	}
	if !m.HasMatch("payment.processed") {
		t.Error("HasMatch(payment.processed) should be true")
	}
	if m.HasMatch("user.created") {
		t.Error("HasMatch(user.created) should be false")
	}
}

// TestTrieMatcherAddRemoveRefcount 验证 Remove 引用计数和节点清理。
func TestTrieMatcherAddRemoveRefcount(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("a.b")
	m.Add("a.b") // refCount = 2

	m.Remove("a.b")
	if !m.HasMatch("a.b") {
		t.Error("after first Remove, pattern should still match (refCount=1)")
	}

	m.Remove("a.b")
	if m.HasMatch("a.b") {
		t.Error("after second Remove, pattern should not match (refCount=0)")
	}
}

// TestTrieMatcherRemoveNonExistent 验证删除不存在的 pattern 不崩溃。
func TestTrieMatcherRemoveNonExistent(t *testing.T) {
	m := NewTrieMatcher()
	m.Remove("does.not.exist")
}

// TestTrieMatcherPutNil 验证 Put(nil) 不崩溃。
func TestTrieMatcherPutNil(t *testing.T) {
	m := NewTrieMatcher()
	m.Put(nil)
}

// TestTrieMatcherNoMatch 验证无任何 pattern 时 Match 返回空。
func TestTrieMatcherNoMatch(t *testing.T) {
	m := NewTrieMatcher()
	sp := m.Match("anything")
	if len(*sp) != 0 {
		t.Errorf("empty matcher: Match = %v, want []", *sp)
	}
	m.Put(sp)
}

// TestTrieMatcherCacheInvalidation 验证 Add 后缓存失效。
// 这里只用通配符 pattern，避免触发 exact 快速路径混淆测试逻辑。
func TestTrieMatcherCacheInvalidation(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("x.*")

	// 预热缓存（x.y 不是精确 pattern，走 trie 慢路径 + 写 cache）
	sp := m.Match("x.y")
	count1 := len(*sp)
	m.Put(sp)
	if count1 != 1 {
		t.Fatalf("before Add: Match count=%d, want 1", count1)
	}

	// 添加另一个通配符，使缓存失效
	m.Add("x.**")

	// 重新 Match，应命中两个通配符
	sp = m.Match("x.y")
	count2 := len(*sp)
	m.Put(sp)
	if count2 != 2 {
		t.Errorf("after Add invalidates cache: Match count=%d, want 2", count2)
	}
}

// TestTrieMatcherDepthLimit 验证超过 maxTrieDepth 的 pattern 被安全忽略。
func TestTrieMatcherDepthLimit(t *testing.T) {
	m := NewTrieMatcher()
	deep := "a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q" // 17 segments > maxTrieDepth(16)
	m.Add(deep)                                 // should not panic
	sp := m.Match(deep)
	m.Put(sp)
}

// TestTrieMatcherConcurrentAddMatchRemove 高并发 Add/Match/Remove 不崩溃无竞态。
func TestTrieMatcherConcurrentAddMatchRemove(t *testing.T) {
	m := NewTrieMatcher()
	patterns := []string{"user.*", "order.**", "payment.processed", "item.created"}
	for _, p := range patterns {
		m.Add(p)
	}

	var wg sync.WaitGroup
	events := []string{"user.login", "order.paid", "payment.processed", "item.created", "unknown"}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sp := m.Match(events[i%len(events)])
			m.Put(sp)
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			m.Add(fmt.Sprintf("dynamic.%d", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			m.Remove(fmt.Sprintf("dynamic.%d", i))
		}(i)
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.HasMatch(events[i%len(events)])
		}(i)
	}

	wg.Wait()
}

// TestTrieMatcherPoolReuse 验证 Put 正确归还到 pool，多次 Match+Put 无泄漏。
func TestTrieMatcherPoolReuse(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("evt.*")

	for i := 0; i < 1000; i++ {
		sp := m.Match("evt.test")
		if len(*sp) != 1 {
			t.Fatalf("iter %d: Match count=%d, want 1", i, len(*sp))
		}
		m.Put(sp)
	}
}

// TestTrieMatcherExactCacheHit 精确 pattern 命中 exact sync.Map 快速路径（100次重复）。
func TestTrieMatcherExactCacheHit(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("very.specific.event")

	for i := 0; i < 100; i++ {
		sp := m.Match("very.specific.event")
		if len(*sp) != 1 || (*sp)[0] != "very.specific.event" {
			t.Fatalf("exact cache hit: got %v", *sp)
		}
		m.Put(sp)
	}
}

// TestTrieMatcherRemoveExact 移除精确 pattern 后 Match 不再返回该 pattern。
func TestTrieMatcherRemoveExact(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("remove.me")
	m.Remove("remove.me")

	sp := m.Match("remove.me")
	count := len(*sp)
	m.Put(sp)
	if count != 0 {
		t.Errorf("after Remove: Match count=%d, want 0", count)
	}
}

// TestTrieMatcherHasMatchWildcard HasMatch 通配符慢路径。
func TestTrieMatcherHasMatchWildcard(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("svc.*")

	if !m.HasMatch("svc.started") {
		t.Error("HasMatch(svc.started) with svc.* should be true")
	}
	if m.HasMatch("other.started") {
		t.Error("HasMatch(other.started) with svc.* should be false")
	}
}

// TestTrieMatcherMultiLevelExact 多层路径精确匹配。
func TestTrieMatcherMultiLevelExact(t *testing.T) {
	m := NewTrieMatcher()
	m.Add("a.b.c.d")

	sp := m.Match("a.b.c.d")
	count := len(*sp)
	m.Put(sp)
	if count != 1 {
		t.Errorf("multi-level exact: count=%d, want 1", count)
	}

	sp = m.Match("a.b.c.e")
	count = len(*sp)
	m.Put(sp)
	if count != 0 {
		t.Errorf("multi-level exact miss: count=%d, want 0", count)
	}
}

// ─── 辅助函数 ───

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
