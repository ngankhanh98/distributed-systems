package main

import (
	"log"
	"net"
	"net/rpc"
	"github.com/marcelloh/fastdb"
	"encoding/json"
)


type KeyValue struct {
	db *fastdb.DB
}

type KeyValueArgs struct {
	ID   int
	UUID string
	Text string
}

func (kv *KeyValue) GetValue(request int, reply *string) error {
	value, ok := kv.db.Get("key-value.db", request)

	if ok {
		*reply = string(value)
	}
	
	return nil
}

func (kv *KeyValue) SetValue(record *KeyValueArgs, reply *string) error {
	recordData, _ := json.Marshal(record)
	err := kv.db.Set("key-value.db", record.ID, recordData)

	if err != nil {
		log.Fatal("Fail to set key-value", err)
	}

	*reply = "OK!"
	return nil
}

func main() {
	// Init fastDB
	kv := new(KeyValue)
	db, err := fastdb.Open("key-value.db", 100) // "key-value.db" will be create if empty
	kv.db = db

	if err != nil {
		log.Println("Error opening database:", err)
		return
	}
	defer db.Close()
	
	// Register service to RPC
	rpc.RegisterName("KeyValue", kv)

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("Can not create sever because:", err)
	}
    
  log.Print("Sever is listening on port 1234")

	// Allow RPC accept call from Client
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Accept error:", err)
		}

		go rpc.ServeConn(conn)
	}
}

