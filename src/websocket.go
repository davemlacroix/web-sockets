package main

import (
	"fmt"
)

func main() {
	fmt.Println("Starting websockets client...")
	//get test cases count by connecting to:
	//ws://localhost:9001/getCaseCount

	c := NewClient("127.0.0.1:9001")
	c.Connect()
}
