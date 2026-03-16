package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
)

func main() {
	fmt.Println("Starting websockets client...")

	conn := NewWSClient("127.0.0.1:9001")

	n, err := GetTestCount(conn)
	if err != nil {
		fmt.Println("error reading in number of tests")
		log.Fatal(err)
	}

	fmt.Println("test count: ", n)

	//loop through all test cases and initiate a connection

	err = UpdateReports(conn)
	if err != nil {
		fmt.Println("error updating reports")
		log.Fatal(err)
	}
}

func GetTestCount(conn *WSClient) (int, error) {
	err := conn.Connect("/getCaseCount")
	defer conn.Close()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	message, err := conn.NextMessage()
	if err != nil {
		return 0, err
	}

	if message.Type() != Text {
		return 0, err
	}

	countText, err := message.ReadText()
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(countText)
	if err != nil {
		return 0, err
	}

	message, err = conn.NextMessage()
	if err != nil {
		return 0, err
	}

	if message.Type() != Close {
		return 0, errors.New("expected close message opcode")
	}

	return count, nil
}

func UpdateReports(conn *WSClient) error {
	err := conn.Connect("/updateReports?agent=MyWSClient")
	defer conn.Close()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	message, err := conn.NextMessage()
	if err != nil {
		return err
	}

	if message.Type() != Close {
		return errors.New("expected close message opcode")
	}

	return nil
}
