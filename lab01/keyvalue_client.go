package main

import (
	"fmt"
	"log"
	"net/rpc"
)

type KeyValueArgs struct {
	ID   int
	UUID string
	Text string
}


func main() {
	client, err := rpc.Dial("tcp", "localhost:1234")
	var reply string

	// Set 
	record := &KeyValueArgs{
		ID:    2,
		UUID:  "UUIDtext_",
		Text: "test@example.com",
	}
	err = client.Call("KeyValue.SetValue", record, &reply) // Expect: OK!

	// Get
	err = client.Call("KeyValue.GetValue", 2, &reply) // Expect: {"ID":2,"UUID":"UUIDtext_","Text":"test@example.com"}

	
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(reply)
}

