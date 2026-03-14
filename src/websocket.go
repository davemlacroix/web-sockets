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
	defer c.Close()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	frame, err := c.ReadFrame()
	if err != nil {
		fmt.Println("error reading frame")
		log.Fatal(err)
	}

	if frame.Type() != 1 {
		fmt.Println("unexpected frame type")
		log.Fatal(err)
	}
	count, err := frame.ReadText()
	fmt.Println("test count: ", count)

	frame, err = c.ReadFrame()
}
