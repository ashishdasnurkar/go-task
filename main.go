package main

import (
	"fmt"
	"os"
	"strings"
)

var tasks []string

func addTask(task string) {
	tasks = append(tasks, task)
	fmt.Println("Added task:", task)
}

func list() {
	for i, task := range tasks {
		fmt.Println(i+1, task)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <COMMAND>")
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "add" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go add <TASK>")
			os.Exit(1)
		}

		task := strings.Join(os.Args[2:], " ")
		addTask(task)

		list()
	}
}
