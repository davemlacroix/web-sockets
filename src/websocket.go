package main

import (
	"errors"
	"fmt"
	"io"
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

	// RunTest(conn, 45, agentName)
	//RunTest(conn, 47, agentName)

	for i := 1; i <= n; i++ {
		RunTest(conn, i, agentName)
	}

	err2 := UpdateReports(conn, agentName)
	if err2 != nil {
		fmt.Println("error updating reports")
		log.Fatal(err2)
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
			body, err := io.ReadAll(message)
			// fmt.Println(message.Type())
			// fmt.Println(body)

			if err != nil && err != io.EOF {
				fmt.Println("error with test " + strconv.Itoa(n) + ": " + err.Error())
				break
			}

			SendMessage(conn.conn, message.Type(), body)

			if err == io.EOF {
				break
			}
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
