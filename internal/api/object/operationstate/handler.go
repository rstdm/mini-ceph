package operationstate

import (
	"errors"
	"go.uber.org/zap"
	"sync"
)

var (
	ErrOperationNotAllowed   = errors.New("operation not allowed")
	ErrDeletionMustBeDelayed = errors.New("delete operation must be delayed until all reads are done")
)

type operationState struct {
	// a process wants to create an object and is currently checking the preconditions.
	wantCreate bool
	// a process wants to delete an object and is currently checking the preconditions.
	wantDelete bool
	// the object is currently created. It must not be read or deleted
	creating bool
	// A read attempt for this object is currently executed. The attempt can fail if the object doesn't exist.
	// All delete operations must be delayed until the read attempts are completed.
	reading int32
	// the object is currently deleted or will be deleted as soon as the last read operation finishes. New read
	// operations must not be allowed. This callback will be called when the last read operation finishes.
	deleting func()
}

type Handler struct {
	mu           sync.Mutex
	objectStates map[string]operationState
	sugar        *zap.SugaredLogger
}

func NewHandler(sugar *zap.SugaredLogger) *Handler {
	return &Handler{
		objectStates: map[string]operationState{},
		sugar:        sugar,
	}
}

// Create

func (h *Handler) WantCreate(objectHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	if state.creating || state.wantCreate {
		return ErrOperationNotAllowed
	}

	state.wantCreate = true
	h.setState(objectHash, state)

	return nil
}

func (h *Handler) AbortWantCreate(objectHash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.wantCreate = false
	h.setState(objectHash, state)
}

func (h *Handler) StartCreating(objectHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.wantCreate = false

	// we don't have to check for active read operations because read (precondition: file exists) and
	// create (precondition: file doesn't exist) are mutual exclusive
	if state.creating || state.deleting != nil {
		h.setState(objectHash, state)
		return ErrOperationNotAllowed
	}

	state.creating = true
	h.setState(objectHash, state)

	return nil
}

func (h *Handler) DoneCreating(objectHash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.creating = false
	h.setState(objectHash, state)
}

// Read

func (h *Handler) StartReading(objectHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	if state.creating || state.deleting != nil {
		return ErrOperationNotAllowed
	}

	state.reading += 1
	h.setState(objectHash, state)

	return nil
}

func (h *Handler) DoneReading(objectHash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.reading -= 1
	if state.reading < 0 {
		h.sugar.Warnw("The read operation counter for an object has a negative value. This indicates that not "+
			"every read operation has been marked as completed.", ""+
			"objectHash", objectHash,
			"readCounter", state.reading,
		)
		state.reading = 0
	}

	if state.reading == 0 && state.deleting != nil {
		// there is a pending delete operation which must be resumed
		// the deletion has to be executed in its own goroutine. Otherwise, it would keep the mutex locked
		go state.deleting()
	}

	h.setState(objectHash, state)
}

// Delete

func (h *Handler) WantDelete(objectHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	if state.deleting != nil || state.wantDelete {
		return ErrOperationNotAllowed
	}

	state.wantDelete = true
	h.setState(objectHash, state)

	return nil
}

func (h *Handler) AbortWantDelete(objectHash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.wantDelete = false
	h.setState(objectHash, state)
}

func (h *Handler) StartDeleting(objectHash string, deleteCallback func()) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.wantDelete = false
	if state.creating || state.deleting != nil {
		h.setState(objectHash, state)
		return ErrOperationNotAllowed
	}

	// set deleting to true. This will prevent further create, read and delete operations
	// The actual deletion will be delayed until all read operations are done
	state.deleting = deleteCallback
	h.setState(objectHash, state)

	if state.reading > 0 {
		return ErrDeletionMustBeDelayed
	}

	return nil
}

func (h *Handler) DoneDeleting(objectHash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	state.deleting = nil
	h.setState(objectHash, state)
}

// Helper functions

func (h *Handler) setState(objectHash string, state operationState) {
	operationsRunning := state.wantCreate || state.wantDelete || state.creating || state.reading > 0 || state.deleting != nil
	if operationsRunning {
		h.objectStates[objectHash] = state

	} else {
		delete(h.objectStates, objectHash)
	}
}

func (h *Handler) getState(objectHash string) operationState {
	state, ok := h.objectStates[objectHash]
	if !ok { // if the object hash is not in the map then no transaction is running
		return operationState{} // all fields have their default value
	}

	return state
}
