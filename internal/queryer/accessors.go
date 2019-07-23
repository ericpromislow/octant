package queryer

import (
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/vmware/octant/pkg/store"
)

type childrenCache struct {
	children map[types.UID][]runtime.Object
	mu       sync.RWMutex
}

func initChildrenCache() *childrenCache {
	return &childrenCache{
		children: make(map[types.UID][]runtime.Object),
	}
}

func (c *childrenCache) get(key types.UID) ([]runtime.Object, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.children[key]
	return v, ok
}

func (c *childrenCache) set(key types.UID, value []runtime.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.children[key] = value
}

type ownerCache struct {
	owner map[store.Key]runtime.Object
	mu    sync.Mutex
}

func initOwnerCache() *ownerCache {
	return &ownerCache{
		owner: make(map[store.Key]runtime.Object),
	}
}

func (c *ownerCache) set(key store.Key, value runtime.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if value == nil {
		return
	}

	c.owner[key] = value
}

func (c *ownerCache) get(key store.Key) (runtime.Object, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.owner[key]
	return v, ok
}

type podsForServicesCache struct {
	podsForServices map[types.UID][]*v1.Pod
	mu              sync.Mutex
}

func initPodsForServicesCache() *podsForServicesCache {
	return &podsForServicesCache{
		podsForServices: make(map[types.UID][]*v1.Pod),
	}
}

func (c *podsForServicesCache) set(key types.UID, value []*v1.Pod) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.podsForServices[key] = value
}

func (c *podsForServicesCache) get(key types.UID) ([]*v1.Pod, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.podsForServices[key]
	return v, ok
}
