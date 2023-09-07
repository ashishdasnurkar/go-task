package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var tasks []string
var db *sql.DB

func addTask(task string) {
	//tasks = append(tasks, task)
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

func list() {

	rows, err := db.Query("select description from tasks")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var desc string
		err = rows.Scan(&desc)
		if err != nil {
			log.Fatal(err)
		}
		tasks = append(tasks, desc)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	for i, task := range tasks {
		fmt.Println(i+1, task)
	}
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
	default:
		showUsage(1)
	}
}
