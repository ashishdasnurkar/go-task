package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type taskType struct {
	Id          string    `json:"uuid"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"createdAt"   db:"created_at"`
}

func execStatement(stmtStr string, args ...interface{}) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(stmtStr)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func addTask(task string) {
	stmtStr := "insert into tasks (uuid, description, done) values(?, ?, ?)"
	id := uuid.New().String()

	err := execStatement(stmtStr, id, task, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Added task:", task)
}

func updateTask(task taskType) {
	stmtStr := "update tasks set description = ? where uuid = ?"
	err := execStatement(stmtStr, task.Description, task.Id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Task updated")
}

func getTasks(whereStr string) []taskType {
	var tasks []taskType

	rows, err := db.Query("select uuid, description, done from tasks " + whereStr)
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

func getAllIncompleteTasks() []taskType {
	return getTasks("where done = false ORDER BY created_at ASC")
}

func getAllCompletedtasks() []taskType {
	return getTasks("where done = true")
}

func getTaskByIndex(index int) (taskType, error) {
	var task taskType

	if index <= 0 {
		return task, errors.New("Index out of bound")
	}

	tasks := getAllIncompleteTasks()
	if len(tasks) < index {
		return task, errors.New("Index out of bound")
	}
	task = tasks[index-1]

	return task, nil
}

func list() {
	tasks := getAllIncompleteTasks()

	for i, task := range tasks {
		fmt.Println(i+1, task.Description)
	}
}

func listCompleted() {
	tasks := getAllCompletedtasks()

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

	stmtStr := "update tasks set done = true where uuid = ?"
	err = execStatement(stmtStr, task.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Marked as done:", task.Description)
}

func deleteTask(id int) {
	task, err := getTaskByIndex(id)
	if err != nil {
		fmt.Println("Invalid ID")
		os.Exit(1)
	}
	stmtStr := "delete from tasks where uuid = ?"
	err = execStatement(stmtStr, task.Id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Deleted: ", task.Description)
}

func editTask(id int) {
	task, err := getTaskByIndex(id)
	if err != nil {
		log.Fatal(err)
	}

	// create string for the tmp file
	templateStr := `
# UUID:	%s
# ID:	%s
  Description: %s
`
	outputStr := fmt.Sprintf(templateStr, task.Id, strconv.Itoa(id), task.Description)

	fmt.Println(outputStr)
	// write this outputStr to a tmp file
	tmpFileName := "./tmpTask.task"
	err = os.WriteFile(tmpFileName, []byte(outputStr), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// open tempTask.task file in default editor

	// Get the EDITOR environment variable.
	editor := os.Getenv("EDITOR")

	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpFileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Read the tmpFile back
	dat, err := os.ReadFile(tmpFileName)
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(dat), "\n")
	hasUpdates := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Ignore blank lines
		if len(trimmed) == 0 {
			continue
		}
		// Ignore comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.Split(trimmed, ":")
		if len(parts) < 2 {
			log.Fatalf("Invalid field: %s", trimmed)
		}
		field := parts[0]
		value := strings.TrimSpace(parts[1])

		if field == "Description" {
			if task.Description == value {
				continue
			}
			task.Description = value
			hasUpdates = true
			fmt.Println("Updating Description")
		} else {
			log.Fatalf("Invalid updat field: %s", field)
		}
	}

	if !hasUpdates {
		fmt.Println("Nothing to update")
		os.Exit(0)
	}

	updateTask(task)
}

func addToDescription(id int, descriptionStr string, isAppend bool) {
	task, err := getTaskByIndex(id)
	if err != nil {
		log.Fatal(err)
	}

	if isAppend {
		task.Description = fmt.Sprintf("%s %s", task.Description, descriptionStr)
		fmt.Printf("Appending to task %d '%s'\n", id, task.Description)
	} else {
		task.Description = fmt.Sprintf("%s %s", descriptionStr, task.Description)
		fmt.Printf("Prepending to task %d '%s'\n", id, task.Description)
	}

	updateTask(task)
}

func exportTasks() {
	tasks := getTasks("")

	jsonData, err := json.MarshalIndent(tasks, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonData))
}

func makeHttpCall(data []byte) {
	url := "http://localhost:8080/sync"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}

	fmt.Println("response Body:", string(body))
}

func syncTasks() {
	tasks := getTasks("")
	var ts []*TaskType

	for _, task := range tasks {
		t := &TaskType{
			Id:          task.Id,
			Description: task.Description,
			Done:        task.Done,
			CreatedAt:   task.CreatedAt.Unix(),
		}
		ts = append(ts, t)
	}

	taskList := &TaskList{
		Tasks: ts,
	}

	// Marshal the TaskList into a byte array
	data, err := proto.Marshal(taskList)
	if err != nil {
		log.Fatal("Marshaling error: ", err)
	}

	// Output the Base64 string
	//fmt.Println("Base64 Encoded Protobuf Data: ", base64Data)

	makeHttpCall(data)
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
    done BOOLEAN,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

		descriptionStr := strings.Join(os.Args[2:], " ")
		addTask(descriptionStr)
		list()
	case "list":
		list()
	case "done":
		markDone(id)
	case "delete":
		deleteTask(id)
	case "edit":
		editTask(id)
	case "append":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go add <TASK>")
			os.Exit(1)
		}

		descriptionStr := strings.Join(os.Args[3:], " ")
		addToDescription(id, descriptionStr, true)
	case "prepend":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go add <TASK>")
			os.Exit(1)
		}

		descriptionStr := strings.Join(os.Args[3:], " ")
		addToDescription(id, descriptionStr, false)
	case "completed":
		listCompleted()
	case "export":
		exportTasks()
	case "sync":
		syncTasks()
	default:
		showUsage(1)
	}
}
