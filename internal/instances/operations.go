package instances

import (
	"context"
	"fmt"
	"sync"
)

// OperationType identifies the type of operation.
type OperationType string

const (
	OpDownload OperationType = "download"
	OpLaunch   OperationType = "launch"
)

// OperationStatus is the current status of an operation.
type OperationStatus string

const (
	OpStatusRunning   OperationStatus = "running"
	OpStatusCompleted OperationStatus = "completed"
	OpStatusFailed    OperationStatus = "failed"
	OpStatusCancelled OperationStatus = "cancelled"
)

// Operation represents an active or completed operation on an instance.
type Operation struct {
	ID            string
	InstanceID    string
	Type          OperationType
	Status        OperationStatus
	CancelContext context.Context
	CancelFunc    context.CancelFunc
}

// Registry tracks active operations per instance.
type Registry struct {
	mu    sync.Mutex
	ops   map[string]*Operation // keyed by instance ID
}

// NewRegistry creates a new operation registry.
func NewRegistry() *Registry {
	return &Registry{
		ops: make(map[string]*Operation),
	}
}

// Start begins a new operation for an instance.
// Returns error if an operation is already active for that instance.
func (r *Registry) Start(instanceID string, opType OperationType) (*Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.ops[instanceID]; ok {
		if existing.Status == OpStatusRunning {
			return nil, fmt.Errorf("operation already active for instance %s", instanceID)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	op := &Operation{
		ID:            instanceID + "-" + string(opType),
		InstanceID:    instanceID,
		Type:          opType,
		Status:        OpStatusRunning,
		CancelContext: ctx,
		CancelFunc:    cancel,
	}

	r.ops[instanceID] = op
	return op, nil
}

// Complete marks an operation as completed.
func (r *Registry) Complete(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if op, ok := r.ops[instanceID]; ok {
		op.Status = OpStatusCompleted
	}
}

// Fail marks an operation as failed.
func (r *Registry) Fail(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if op, ok := r.ops[instanceID]; ok {
		op.Status = OpStatusFailed
	}
}

// Cancel cancels an active operation.
func (r *Registry) Cancel(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if op, ok := r.ops[instanceID]; ok && op.Status == OpStatusRunning {
		op.CancelFunc()
		op.Status = OpStatusCancelled
	}
}

// Get returns the operation for an instance, if any.
func (r *Registry) Get(instanceID string) *Operation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ops[instanceID]
}

// IsActive checks if an operation is currently active for an instance.
func (r *Registry) IsActive(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	op, ok := r.ops[instanceID]
	return ok && op.Status == OpStatusRunning
}
