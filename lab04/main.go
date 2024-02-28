package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/cespare/xxhash"
)

// Hash is responsible for generating unsigned, 64-bit hash of provided byte slice.
// Hash should minimize collisions (generating same hash for different byte slice)
// and while performance is also important fast functions are preferable (i.e.
// you can use FarmHash family).
type Hash interface {
	Sum64([]byte) uint64
}

// Node interface represents a key in consistent hash ring.
type Node interface {
	String() string
}

// Config represents a structure to control consistent package.
type Config struct {
	// Hash is responsible for generating unsigned, 64-bit hash of provided byte slice.
	Hash Hash

	// Keys are distributed among keys. Prime numbers are good to
	// distribute keys uniformly. Select a big KeyCount if you have
	// too many keys.
	KeyCount int

	// Keyss are replicated on consistent hash ring. This number means that a key
	// how many times replicated on the ring.
	ReplicationFactor int

	// Load is used to calculate average load. See the code, the paper and Google's blog post to learn about it.
	Load float64
}

// Consistent holds the information about the nodes of the consistent hash circle.
type Consistent struct {
	mu sync.RWMutex

	config    Config
	hash      Hash
	sortedSet []uint64
	keyCount  uint64
	loads     map[string]float64
	nodes     map[string]*Node
	keys      map[int]*Node
	ring      map[uint64]*Node
}

type hash struct{}

func (h hash) Sum64(data []byte) uint64 {
	// you should use a proper hash function for uniformity.
	return xxhash.Sum64(data)
}

// New creates and returns a new Consistent object.
func New(nodes []Node, config Config) *Consistent {
	if config.Hash == nil {
		panic("Hash cannot be nil")
	}

	c := &Consistent{
		config:   config,
		nodes:    make(map[string]*Node),
		keyCount: uint64(config.KeyCount),
		ring:     make(map[uint64]*Node),
	}

	c.hash = config.Hash
	for _, key := range nodes {
		c.add(key)
	}
	if nodes != nil {
		c.distributeKeys()
	}
	return c
}

// AddNode adds a new key to the consistent hash circle.
func (c *Consistent) AddNode(node Node) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.nodes[node.String()]; ok {
		return
	}
	c.add(node)
	c.distributeKeys()
}

func (c *Consistent) RemoveNode(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.nodes[name]; !ok {
		// There is no key with that name. Quit immediately.
		return
	}

	for i := 0; i < c.config.ReplicationFactor; i++ {
		hKey := []byte(fmt.Sprintf("%s%d", name, i))
		h := c.hash.Sum64(hKey)
		delete(c.ring, h)
		c.deleteSlice(h)
	}
	delete(c.nodes, name)
	if len(c.nodes) == 0 {
		// consistent hash ring is empty now. Reset the key table.
		c.keys = make(map[int]*Node)
		return
	}
	c.distributeKeys()
}

func (c *Consistent) GetNodeId(key []byte) Node {
	keyID := c.findKeyID(key)
	return c.getKeyNode(keyID)
}

func (c *Consistent) averageLoad() float64 {
	if len(c.nodes) == 0 {
		return 0
	}

	avgLoad := float64(c.keyCount/uint64(len(c.nodes))) * c.config.Load
	return math.Ceil(avgLoad)
}

func (c *Consistent) distributeWithLoad(keyID, idx int, keys map[int]*Node, loads map[string]float64) {
	avgLoad := c.averageLoad()
	var count int
	for {
		count++
		if count >= len(c.sortedSet) {
			// User needs to decrease key count, increase key count or increase load factor.
			panic("not enough room to distribute keys")
		}
		i := c.sortedSet[idx]
		key := *c.ring[i]
		load := loads[key.String()]
		if load+1 <= avgLoad {
			keys[keyID] = &key
			loads[key.String()]++
			return
		}
		idx++
		if idx >= len(c.sortedSet) {
			idx = 0
		}
	}
}

func (c *Consistent) distributeKeys() {
	loads := make(map[string]float64)
	keys := make(map[int]*Node)

	bs := make([]byte, 8)
	for keyID := uint64(0); keyID < c.keyCount; keyID++ {
		binary.LittleEndian.PutUint64(bs, keyID)
		h := c.hash.Sum64(bs)
		idx := sort.Search(len(c.sortedSet), func(i int) bool {
			return c.sortedSet[i] >= h
		})
		if idx >= len(c.sortedSet) {
			idx = 0
		}
		c.distributeWithLoad(int(keyID), idx, keys, loads)
	}
	c.keys = keys
	c.loads = loads
}

func (c *Consistent) add(node Node) {
	for i := 0; i < c.config.ReplicationFactor; i++ {
		hLKey := []byte(fmt.Sprintf("%s%d", node.String(), i))
		h := c.hash.Sum64(hLKey)
		c.ring[h] = &node
		c.sortedSet = append(c.sortedSet, h)
	}
	// sort hashes ascendingly
	sort.Slice(c.sortedSet, func(i int, j int) bool {
		return c.sortedSet[i] < c.sortedSet[j]
	})
	// Storing key at this map is useful to find backup nodes of a key.
	c.nodes[node.String()] = &node
}

func (c *Consistent) deleteSlice(val uint64) {
	for i := 0; i < len(c.sortedSet); i++ {
		if c.sortedSet[i] == val {
			c.sortedSet = append(c.sortedSet[:i], c.sortedSet[i+1:]...)
			break
		}
	}
}

func (c *Consistent) findKeyID(hKey []byte) int {
	h := c.hash.Sum64(hKey)
	return int(h % c.keyCount)
}

func (c *Consistent) getKeyNode(keyID int) Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, ok := c.keys[keyID]
	if !ok {
		return nil
	}
	return *key
}

type myNode string

func (m myNode) String() string {
	return string(m)
}

func main() {
	cfg := Config{
		KeyCount:          7,
		ReplicationFactor: 20,
		Load:              1.25,
		Hash:              hash{},
	}
	c := New(nil, cfg)

	node1 := myNode("1")
	c.AddNode(node1)

	node2 := myNode("80")
	c.AddNode(node2)

	// Create an array to copy old nodes
	nodes := make(map[int]string)
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("%v", i))
		nodes[i] = c.GetNodeId(key).String()
	}

	// Add new node
	node3 := myNode("34")
	c.AddNode(node3)

	// Create an array to compare to the list old nodes
	fmt.Printf("Distribute keys after adding new node\n")
	addedNodes := make(map[int]string)
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("%v", i))
		addedNodes[i] = c.GetNodeId(key).String()
		fmt.Printf("%v: %s => %s\n", i, nodes[i], addedNodes[i])
	}

	// Remove node3
	fmt.Printf("Distribute keys after removing node\n")
	c.RemoveNode(node3.String())

	// Compare to the list old nodes
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("%v", i))
		newNode := c.GetNodeId(key).String()
		fmt.Printf("%v: %s => %s\n", i, addedNodes[i], newNode)
	}
}
