package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <COMMAND>")
		os.Exit(1)
	}

	command := os.Args[1]

	fmt.Println("Command:", command)
	fmt.Println("Task:", strings.Join(os.Args[2:], " "))
}
