# 

## Installation
Here is how to install consistent hash, with 3 additional methods: AddNode, RemoveNode, GetNodeId(key)

<strong>1. Install a Hash package</strong>
```bash
$ go get github.com/cespare/xxhash
```

<strong>2. Import packages in Your Go Code </strong>
```go
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/cespare/xxhash"
)
```

<strong>3. Create a new Ring (Consistent Hash) </strong>

```go
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
```

<strong>4. Add Node </strong>
```go
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
```

<strong>6. Remove node </strong>
```go
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
```

<strong>7. Test how it re-distribute key after adding new Node </strong>
```go
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
}
```

<strong>8. Test how it re-distribute key after removing a Node </strong>
```go
func main() {
	// Remove node3
	c.RemoveNode(node3.String())

	// Compare to the list old nodes
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("%v", i))
		newNode := c.GetNodeId(key).String()
		fmt.Printf("%v: %s => %s\n", i, addedNodes[i], newNode)
	}
}
```


<strong>9. Run</strong>
```bash
# Start the consistent hash 
$ go run main.go
```

## Show case
![Screenshot 2023-12-31 223426](https://github.com/ngankhanh98/distributed-systems/assets/32817908/34dd3da0-9737-4bab-912e-0e2d3b319d8e)
