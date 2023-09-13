package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type taskType struct {
	Id          string
	Description string
	Done        bool
}

func addTask(task string) {
	// tasks = append(tasks, task)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	id := uuid.New().String()

	stmt, err := tx.Prepare("insert into tasks (uuid, description, done) values(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(id, task, false)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Added task:", task)
}

func getAllTasks() []taskType {
	var tasks []taskType

	rows, err := db.Query("select uuid, description, done from tasks")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, desc string
		var done bool
		err = rows.Scan(&id, &desc, &done)
		if err != nil {
			log.Fatal(err)
		}
		tasks = append(tasks, taskType{Id: id, Description: desc, Done: done})
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return tasks
}

func getTaskByIndex(index int) (taskType, error) {
	var task taskType

	if index <= 0 {
		return task, errors.New("Index out of bound")
	}

	tasks := getAllTasks()
	if len(tasks) < index {
		return task, errors.New("Index out of bound")
	}
	task = tasks[index-1]

	return task, nil
}

func list() {
	tasks := getAllTasks()

	for i, task := range tasks {
		fmt.Println(i+1, task.Description)
	}
}

func markDone(id int) {
	task, err := getTaskByIndex(id)
	if err != nil {
		fmt.Println("Invalid ID")
		os.Exit(1)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("update tasks set done = true where uuid = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(task.Id)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Marked as done:", task.Description)
}

func showUsage(returnCode int) {
	fmt.Println("Usage: go run main.go <COMMAND>")
	os.Exit(returnCode)
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./tasks.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	// tasks table fields uuid, description, done

	sqlStmt := `
CREATE TABLE IF NOT EXISTS tasks (
    uuid TEXT PRIMARY KEY,
    description TEXT,
    done BOOLEAN
);
`
	_, err = db.Exec(sqlStmt)

	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		showUsage(1)
	}

	command := os.Args[1]

	id, err := strconv.Atoi(command)
	if err == nil {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go <TASK_ID> <COMMAND>")
			os.Exit(1)
		}
		command = os.Args[2]
	}

	switch command {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go add <TASK>")
			os.Exit(1)
		}

		task := strings.Join(os.Args[2:], " ")
		addTask(task)
		list()
	case "list":
		list()
	case "done":
		markDone(id)
	default:
		showUsage(1)
	}
}
