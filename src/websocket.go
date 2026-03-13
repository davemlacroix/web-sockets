package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Starting websockets client...")
	//get test cases count by connecting to:
	//ws://localhost:9001/getCaseCount

	c := NewWSClient("127.0.0.1:9001")
	err := c.Connect()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	defer c.Close()
}
