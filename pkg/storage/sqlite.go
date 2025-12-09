package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS goals (
		id TEXT PRIMARY KEY,
		description TEXT NOT NULL,
		state TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME,
		result TEXT,
		error TEXT
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		goal_id TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL,
		dependencies TEXT,
		tool_name TEXT,
		tool_args TEXT,
		result TEXT,
		error TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		FOREIGN KEY (goal_id) REFERENCES goals(id)
	);

	CREATE TABLE IF NOT EXISTS execution_logs (
		id TEXT PRIMARY KEY,
		goal_id TEXT NOT NULL,
		task_id TEXT,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		data TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (goal_id) REFERENCES goals(id)
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_goal_id ON tasks(goal_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_logs_goal_id ON execution_logs(goal_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateGoal implements Storage.CreateGoal
func (s *SQLiteStorage) CreateGoal(goal *Goal) error {
	query := `INSERT INTO goals (id, description, state, created_at, updated_at, completed_at, result, error)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		goal.ID,
		goal.Description,
		goal.State,
		goal.CreatedAt,
		goal.UpdatedAt,
		goal.CompletedAt,
		goal.Result,
		goal.Error,
	)
	return err
}

// GetGoal implements Storage.GetGoal
func (s *SQLiteStorage) GetGoal(id string) (*Goal, error) {
	query := `SELECT id, description, state, created_at, updated_at, completed_at, result, error
	          FROM goals WHERE id = ?`

	goal := &Goal{}
	err := s.db.QueryRow(query, id).Scan(
		&goal.ID,
		&goal.Description,
		&goal.State,
		&goal.CreatedAt,
		&goal.UpdatedAt,
		&goal.CompletedAt,
		&goal.Result,
		&goal.Error,
	)

	if err != nil {
		return nil, err
	}
	return goal, nil
}

// UpdateGoal implements Storage.UpdateGoal
func (s *SQLiteStorage) UpdateGoal(goal *Goal) error {
	query := `UPDATE goals SET description = ?, state = ?, updated_at = ?, 
	          completed_at = ?, result = ?, error = ? WHERE id = ?`

	_, err := s.db.Exec(query,
		goal.Description,
		goal.State,
		goal.UpdatedAt,
		goal.CompletedAt,
		goal.Result,
		goal.Error,
		goal.ID,
	)
	return err
}

// ListGoals implements Storage.ListGoals
func (s *SQLiteStorage) ListGoals() ([]*Goal, error) {
	query := `SELECT id, description, state, created_at, updated_at, completed_at, result, error
	          FROM goals ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []*Goal
	for rows.Next() {
		goal := &Goal{}
		err := rows.Scan(
			&goal.ID,
			&goal.Description,
			&goal.State,
			&goal.CreatedAt,
			&goal.UpdatedAt,
			&goal.CompletedAt,
			&goal.Result,
			&goal.Error,
		)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

// CreateTask implements Storage.CreateTask
func (s *SQLiteStorage) CreateTask(task *Task) error {
	depsJSON, _ := json.Marshal(task.Dependencies)
	argsJSON, _ := json.Marshal(task.ToolArgs)

	query := `INSERT INTO tasks (id, goal_id, description, status, dependencies, tool_name, 
	          tool_args, result, error, created_at, updated_at, attempts, max_attempts)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		task.ID,
		task.GoalID,
		task.Description,
		task.Status,
		string(depsJSON),
		task.ToolName,
		string(argsJSON),
		task.Result,
		task.Error,
		task.CreatedAt,
		task.UpdatedAt,
		task.Attempts,
		task.MaxAttempts,
	)
	return err
}

// GetTask implements Storage.GetTask
func (s *SQLiteStorage) GetTask(id string) (*Task, error) {
	query := `SELECT id, goal_id, description, status, dependencies, tool_name, tool_args, 
	          result, error, created_at, updated_at, attempts, max_attempts
	          FROM tasks WHERE id = ?`

	task := &Task{}
	var depsJSON, argsJSON string

	err := s.db.QueryRow(query, id).Scan(
		&task.ID,
		&task.GoalID,
		&task.Description,
		&task.Status,
		&depsJSON,
		&task.ToolName,
		&argsJSON,
		&task.Result,
		&task.Error,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.Attempts,
		&task.MaxAttempts,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(depsJSON), &task.Dependencies)
	json.Unmarshal([]byte(argsJSON), &task.ToolArgs)

	return task, nil
}

// UpdateTask implements Storage.UpdateTask
func (s *SQLiteStorage) UpdateTask(task *Task) error {
	depsJSON, _ := json.Marshal(task.Dependencies)
	argsJSON, _ := json.Marshal(task.ToolArgs)

	query := `UPDATE tasks SET description = ?, status = ?, dependencies = ?, 
	          tool_name = ?, tool_args = ?, result = ?, error = ?, updated_at = ?, 
	          attempts = ?, max_attempts = ? WHERE id = ?`

	_, err := s.db.Exec(query,
		task.Description,
		task.Status,
		string(depsJSON),
		task.ToolName,
		string(argsJSON),
		task.Result,
		task.Error,
		task.UpdatedAt,
		task.Attempts,
		task.MaxAttempts,
		task.ID,
	)
	return err
}

// GetTasksByGoal implements Storage.GetTasksByGoal
func (s *SQLiteStorage) GetTasksByGoal(goalID string) ([]Task, error) {
	query := `SELECT id, goal_id, description, status, dependencies, tool_name, tool_args, 
	          result, error, created_at, updated_at, attempts, max_attempts
	          FROM tasks WHERE goal_id = ? ORDER BY created_at ASC`

	rows, err := s.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		task := Task{}
		var depsJSON, argsJSON string

		err := rows.Scan(
			&task.ID,
			&task.GoalID,
			&task.Description,
			&task.Status,
			&depsJSON,
			&task.ToolName,
			&argsJSON,
			&task.Result,
			&task.Error,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.Attempts,
			&task.MaxAttempts,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(depsJSON), &task.Dependencies)
		json.Unmarshal([]byte(argsJSON), &task.ToolArgs)

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// CreateLog implements Storage.CreateLog
func (s *SQLiteStorage) CreateLog(log *ExecutionLog) error {
	query := `INSERT INTO execution_logs (id, goal_id, task_id, level, message, data, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		log.ID,
		log.GoalID,
		log.TaskID,
		log.Level,
		log.Message,
		log.Data,
		log.CreatedAt,
	)
	return err
}

// GetLogsByGoal implements Storage.GetLogsByGoal
func (s *SQLiteStorage) GetLogsByGoal(goalID string) ([]*ExecutionLog, error) {
	query := `SELECT id, goal_id, task_id, level, message, data, created_at
	          FROM execution_logs WHERE goal_id = ? ORDER BY created_at ASC`

	rows, err := s.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*ExecutionLog
	for rows.Next() {
		log := &ExecutionLog{}
		err := rows.Scan(
			&log.ID,
			&log.GoalID,
			&log.TaskID,
			&log.Level,
			&log.Message,
			&log.Data,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// Close implements Storage.Close
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
