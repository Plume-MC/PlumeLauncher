package services

import (
	"plumelauncher/internal/instances"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func transitionInstanceState(app *application.App, manager *instances.Manager, id string, next instances.InstanceState, operationID string) error {
	current, err := manager.Get(id)
	if err != nil {
		return NewNotFoundError("instance not found")
	}
	if current.State == next {
		return nil
	}
	if err := manager.UpdateState(id, next); err != nil {
		return NewConflictError(err.Error())
	}
	emit(app, EventInstanceState, InstanceStateEvent{
		OperationID: operationID,
		InstanceID:  id,
		OldState:    string(current.State),
		NewState:    string(next),
	})
	return nil
}
