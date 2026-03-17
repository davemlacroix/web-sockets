package main

import (
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

	// RunTest(conn, 47, agentName)
	// RunTest(conn, 50, agentName)

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
		mType, err := conn.NextMessage()
		if err != nil && err != io.EOF {
			fmt.Println("error with test " + strconv.Itoa(n) + ": " + err.Error())
			break
		}
		if err == io.EOF {
			break
		}

		body, err := io.ReadAll(conn)
		conn.Write(mType, body)
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

	mType, err := conn.NextMessage()
	if err != nil {
		return 0, err
	}

	body, err := io.ReadAll(conn)
	if mType != Text {
		return 0, err
	}

	countText := string(body)
	fmt.Println(countText)
	count, err := strconv.Atoi(countText)
	if err != nil {
		return 0, err
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
