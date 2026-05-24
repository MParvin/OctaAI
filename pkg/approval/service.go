package approval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mparvin/octaai/pkg/storage"
)

// Service manages human approval workflows.
type Service struct {
	store storage.Storage
}

// NewService creates an approval service.
func NewService(store storage.Storage) *Service {
	return &Service{store: store}
}

// Fingerprint creates a stable hash for a tool invocation.
func Fingerprint(toolName string, args map[string]interface{}) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		if k == "_isolated" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	payload := toolName + ":"
	for _, k := range keys {
		payload += k + "=" + fmt.Sprintf("%v", args[k]) + ";"
	}
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// CreatePending stores a new approval request.
func (s *Service) CreatePending(goalID, taskID, toolName string, args map[string]interface{}, reason string) (*storage.ApprovalRequest, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	req := &storage.ApprovalRequest{
		ID:           fmt.Sprintf("approval_%d", time.Now().UnixNano()),
		GoalID:       goalID,
		TaskID:       taskID,
		ToolName:     toolName,
		ToolArgsJSON: string(argsJSON),
		Fingerprint:  Fingerprint(toolName, args),
		Reason:       reason,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
	if err := s.store.CreateApproval(req); err != nil {
		return nil, err
	}
	return req, nil
}

// Approve marks an approval as approved and resumes the goal.
func (s *Service) Approve(approvalID string) (*storage.ApprovalRequest, error) {
	req, err := s.store.GetApproval(approvalID)
	if err != nil {
		return nil, err
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("approval %s is already %s", approvalID, req.Status)
	}

	now := time.Now()
	req.Status = "approved"
	req.ResolvedAt = &now
	if err := s.store.UpdateApproval(req); err != nil {
		return nil, err
	}

	goal, err := s.store.GetGoal(req.GoalID)
	if err != nil {
		return req, err
	}
	goal.State = storage.StateIdle
	goal.UpdatedAt = now
	if err := s.store.UpdateGoal(goal); err != nil {
		return req, err
	}

	task, err := s.store.GetTask(req.TaskID)
	if err == nil && task.Status == "running" {
		task.Status = "pending"
		task.Error = ""
		task.UpdatedAt = now
		_ = s.store.UpdateTask(task)
	}

	return req, nil
}

// Deny marks an approval as denied and fails the goal.
func (s *Service) Deny(approvalID string) (*storage.ApprovalRequest, error) {
	req, err := s.store.GetApproval(approvalID)
	if err != nil {
		return nil, err
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("approval %s is already %s", approvalID, req.Status)
	}

	now := time.Now()
	req.Status = "denied"
	req.ResolvedAt = &now
	if err := s.store.UpdateApproval(req); err != nil {
		return nil, err
	}

	goal, err := s.store.GetGoal(req.GoalID)
	if err != nil {
		return req, err
	}
	goal.State = storage.StateFailed
	goal.UpdatedAt = now
	goal.CompletedAt = &now
	goal.Error = fmt.Sprintf("action denied: %s", req.Reason)
	_ = s.store.UpdateGoal(goal)

	return req, nil
}

// ListPending returns all pending approval requests.
func (s *Service) ListPending() ([]storage.ApprovalRequest, error) {
	return s.store.ListPendingApprovals()
}

// IsApproved checks whether a tool action has prior approval.
func (s *Service) IsApproved(goalID, toolName string, args map[string]interface{}) (bool, error) {
	return s.store.HasApprovedAction(goalID, Fingerprint(toolName, args))
}
