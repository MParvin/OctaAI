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

	CREATE TABLE IF NOT EXISTS execution_steps (
		id TEXT PRIMARY KEY,
		goal_id TEXT NOT NULL,
		task_id TEXT,
		description TEXT NOT NULL,
		status TEXT NOT NULL,
		input_json TEXT NOT NULL,
		output_json TEXT,
		validation TEXT,
		validation_detail TEXT,
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		checkpoint_id TEXT,
		FOREIGN KEY (goal_id) REFERENCES goals(id)
	);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		goal_id TEXT NOT NULL,
		state TEXT NOT NULL,
		step_index INTEGER DEFAULT 0,
		payload TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (goal_id) REFERENCES goals(id)
	);

	CREATE INDEX IF NOT EXISTS idx_steps_goal_id ON execution_steps(goal_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_goal_id ON checkpoints(goal_id);

	CREATE TABLE IF NOT EXISTS approval_requests (
		id TEXT PRIMARY KEY,
		goal_id TEXT NOT NULL,
		task_id TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		tool_args_json TEXT NOT NULL,
		fingerprint TEXT NOT NULL,
		reason TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		resolved_at DATETIME,
		FOREIGN KEY (goal_id) REFERENCES goals(id)
	);

	CREATE INDEX IF NOT EXISTS idx_approvals_status ON approval_requests(status);
	CREATE INDEX IF NOT EXISTS idx_approvals_goal_id ON approval_requests(goal_id);
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

// CreateStep implements Storage.CreateStep
func (s *SQLiteStorage) CreateStep(step *ExecutionStepRecord) error {
	query := `INSERT INTO execution_steps (id, goal_id, task_id, description, status, input_json, output_json,
	          validation, validation_detail, retry_count, max_retries, started_at, completed_at, created_at, updated_at, checkpoint_id)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		step.ID, step.GoalID, step.TaskID, step.Description, step.Status,
		step.InputJSON, step.OutputJSON, step.Validation, step.ValidationDetail,
		step.RetryCount, step.MaxRetries, step.StartedAt, step.CompletedAt,
		step.CreatedAt, step.UpdatedAt, step.CheckpointID,
	)
	return err
}

// UpdateStep implements Storage.UpdateStep
func (s *SQLiteStorage) UpdateStep(step *ExecutionStepRecord) error {
	query := `UPDATE execution_steps SET description = ?, status = ?, input_json = ?, output_json = ?,
	          validation = ?, validation_detail = ?, retry_count = ?, max_retries = ?,
	          started_at = ?, completed_at = ?, updated_at = ?, checkpoint_id = ? WHERE id = ?`
	_, err := s.db.Exec(query,
		step.Description, step.Status, step.InputJSON, step.OutputJSON,
		step.Validation, step.ValidationDetail, step.RetryCount, step.MaxRetries,
		step.StartedAt, step.CompletedAt, step.UpdatedAt, step.CheckpointID, step.ID,
	)
	return err
}

// GetStepsByGoal implements Storage.GetStepsByGoal
func (s *SQLiteStorage) GetStepsByGoal(goalID string) ([]ExecutionStepRecord, error) {
	query := `SELECT id, goal_id, task_id, description, status, input_json, output_json,
	          validation, validation_detail, retry_count, max_retries, started_at, completed_at,
	          created_at, updated_at, checkpoint_id
	          FROM execution_steps WHERE goal_id = ? ORDER BY created_at ASC`
	rows, err := s.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []ExecutionStepRecord
	for rows.Next() {
		var step ExecutionStepRecord
		err := rows.Scan(
			&step.ID, &step.GoalID, &step.TaskID, &step.Description, &step.Status,
			&step.InputJSON, &step.OutputJSON, &step.Validation, &step.ValidationDetail,
			&step.RetryCount, &step.MaxRetries, &step.StartedAt, &step.CompletedAt,
			&step.CreatedAt, &step.UpdatedAt, &step.CheckpointID,
		)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// CreateCheckpoint implements Storage.CreateCheckpoint
func (s *SQLiteStorage) CreateCheckpoint(cp *CheckpointRecord) error {
	query := `INSERT INTO checkpoints (id, goal_id, state, step_index, payload, created_at)
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, cp.ID, cp.GoalID, cp.State, cp.StepIndex, cp.Payload, cp.CreatedAt)
	return err
}

// GetCheckpoint implements Storage.GetCheckpoint
func (s *SQLiteStorage) GetCheckpoint(id string) (*CheckpointRecord, error) {
	query := `SELECT id, goal_id, state, step_index, payload, created_at FROM checkpoints WHERE id = ?`
	cp := &CheckpointRecord{}
	err := s.db.QueryRow(query, id).Scan(&cp.ID, &cp.GoalID, &cp.State, &cp.StepIndex, &cp.Payload, &cp.CreatedAt)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// GetCheckpointsByGoal implements Storage.GetCheckpointsByGoal
func (s *SQLiteStorage) GetCheckpointsByGoal(goalID string) ([]CheckpointRecord, error) {
	query := `SELECT id, goal_id, state, step_index, payload, created_at FROM checkpoints WHERE goal_id = ? ORDER BY created_at DESC`
	rows, err := s.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cps []CheckpointRecord
	for rows.Next() {
		var cp CheckpointRecord
		if err := rows.Scan(&cp.ID, &cp.GoalID, &cp.State, &cp.StepIndex, &cp.Payload, &cp.CreatedAt); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, nil
}

// CreateApproval implements Storage.CreateApproval
func (s *SQLiteStorage) CreateApproval(req *ApprovalRequest) error {
	query := `INSERT INTO approval_requests (id, goal_id, task_id, tool_name, tool_args_json, fingerprint, reason, status, created_at, resolved_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		req.ID, req.GoalID, req.TaskID, req.ToolName, req.ToolArgsJSON,
		req.Fingerprint, req.Reason, req.Status, req.CreatedAt, req.ResolvedAt,
	)
	return err
}

// GetApproval implements Storage.GetApproval
func (s *SQLiteStorage) GetApproval(id string) (*ApprovalRequest, error) {
	query := `SELECT id, goal_id, task_id, tool_name, tool_args_json, fingerprint, reason, status, created_at, resolved_at
	          FROM approval_requests WHERE id = ?`
	req := &ApprovalRequest{}
	err := s.db.QueryRow(query, id).Scan(
		&req.ID, &req.GoalID, &req.TaskID, &req.ToolName, &req.ToolArgsJSON,
		&req.Fingerprint, &req.Reason, &req.Status, &req.CreatedAt, &req.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// UpdateApproval implements Storage.UpdateApproval
func (s *SQLiteStorage) UpdateApproval(req *ApprovalRequest) error {
	query := `UPDATE approval_requests SET status = ?, resolved_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, req.Status, req.ResolvedAt, req.ID)
	return err
}

// ListPendingApprovals implements Storage.ListPendingApprovals
func (s *SQLiteStorage) ListPendingApprovals() ([]ApprovalRequest, error) {
	query := `SELECT id, goal_id, task_id, tool_name, tool_args_json, fingerprint, reason, status, created_at, resolved_at
	          FROM approval_requests WHERE status = 'pending' ORDER BY created_at ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []ApprovalRequest
	for rows.Next() {
		var req ApprovalRequest
		if err := rows.Scan(
			&req.ID, &req.GoalID, &req.TaskID, &req.ToolName, &req.ToolArgsJSON,
			&req.Fingerprint, &req.Reason, &req.Status, &req.CreatedAt, &req.ResolvedAt,
		); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// HasApprovedAction implements Storage.HasApprovedAction
func (s *SQLiteStorage) HasApprovedAction(goalID, fingerprint string) (bool, error) {
	query := `SELECT COUNT(*) FROM approval_requests WHERE goal_id = ? AND fingerprint = ? AND status = 'approved'`
	var count int
	err := s.db.QueryRow(query, goalID, fingerprint).Scan(&count)
	return count > 0, err
}

// Close implements Storage.Close()
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
