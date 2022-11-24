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
	creating bool
	reading  int32
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

func (h *Handler) StartCreating(objectHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	if state.creating || state.reading > 0 || state.deleting != nil {
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

func (h *Handler) StartDeleting(objectHash string, deleteCallback func()) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getState(objectHash)
	if state.creating || state.deleting != nil {
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
	operationsRunning := state.creating || state.reading == 0 || state.deleting == nil
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
