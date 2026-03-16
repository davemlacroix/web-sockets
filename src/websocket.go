package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
)

func main() {
	fmt.Println("Starting websockets client...")
	agentName := "MyWSClient"
	conn := NewWSClient("127.0.0.1:9001")

	n, err := GetTestCount(conn)
	if err != nil {
		fmt.Println("error reading in number of tests")
		log.Fatal(err)
	}

	fmt.Println("test count: ", n)

	for i := 0; i < n; i++ {
		RunTest(conn, i, agentName)
	}

	err = UpdateReports(conn, agentName)
	if err != nil {
		fmt.Println("error updating reports")
		log.Fatal(err)
	}
}

func RunTest(conn *WSClient, n int, agentName string) error {
	path := "/runCase?case=" + strconv.Itoa(n) + "&agent=" + agentName
	err := conn.Connect(path)
	defer conn.Close()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	for {
		message, err := conn.NextMessage()
		if err != nil {
			fmt.Println("error with test " + strconv.Itoa(n) + ": " + err.Error())
			break
		}

		if message.Type() == Close {
			break
		}

		if message.Type() == Text || message.Type() == Binary {
			body := make([]byte, 4096) //to start only work with frames less than 4096
			l, err := message.Read(body)
			if err != nil {
				fmt.Println("error with test " + strconv.Itoa(n) + ": " + err.Error())
				break
			}

			SendMessage(conn.conn, message.Type(), body[:l])
		}
	}

	return nil
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

func UpdateReports(conn *WSClient, agentName string) error {
	err := conn.Connect("/updateReports?agent=" + agentName)
	defer conn.Close()
	if err != nil {
		fmt.Println("error with initial connection")
		log.Fatal(err)
	}

	return nil
}
