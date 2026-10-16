package aseprite

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPathLockManager_ConcurrentLock(t *testing.T) {
	mgr := NewPathLockManager()
	testPath := "/tmp/test_sprite.aseprite"

	var order []int
	var mu sync.Mutex

	unlock1 := mgr.Lock(testPath)

	go func() {
		unlock2 := mgr.Lock(testPath)
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		unlock2()
	}()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	order = append(order, 1)
	mu.Unlock()
	unlock1()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{1, 2}, order)
}
