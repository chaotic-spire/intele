package intele

import (
	"context"
	"github.com/nlypage/intele/storage"
	"strings"

	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	stateWaitingInput = "waiting_input"
)

// pendingRequest represents a pending input request from a user
type pendingRequest struct {
	mu        sync.Mutex
	message   *tele.Message // the response message
	callback  *tele.Callback
	completed bool // whether the request has been completed
	canceled  bool // whether the request has been canceled (completed will be true in this case)
	callbacks []tele.CallbackEndpoint
}

// InputManager is a manager for input requests
type InputManager struct {
	storage  storage.StateStorage
	requests sync.Map
}

// InputOptions contains options for the input manager
type InputOptions struct {
	// Storage for storing user states (default: in memory)
	Storage storage.StateStorage
}

// NewInputManager creates a new input manager
//
// NOTE:
//   - If no storage is provided, by default storage.MemoryStorage will be used
func NewInputManager(opts InputOptions) *InputManager {
	if opts.Storage == nil {
		opts.Storage = storage.NewMemoryStorage()
	}
	return &InputManager{
		storage: opts.Storage,
	}
}

// MessageHandler returns a OnText handler function for telebot, that you need to set in your bot, for handling input requests
//
// Example: b.Handle(tele.OnText, b.InputManager.Handler())
func (h *InputManager) MessageHandler() tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Message() == nil {
			return nil
		}

		userID := c.Sender().ID

		// Check if we're waiting for input from this user
		state, err := h.storage.Get(userID)
		if err != nil || state != stateWaitingInput {
			return nil
		}

		// Get or create pending request
		value, _ := h.requests.LoadOrStore(userID, &pendingRequest{})
		req := value.(*pendingRequest)

		// Set response and mark as completed
		req.mu.Lock()
		req.message = c.Message()
		req.completed = true
		req.mu.Unlock()

		// Clean up storage
		h.storage.Delete(userID)

		return nil
	}
}

// CallbackHandler returns a OnCallback handler function for telebot, that you need to set in your bot, for handling input requests
func (h *InputManager) CallbackHandler() tele.HandlerFunc {
	return func(c tele.Context) error {
		userID := c.Sender().ID

		// Check if we're waiting for input from this user
		state, err := h.storage.Get(userID)
		if err != nil || state != stateWaitingInput {
			return nil
		}

		// Get or create pending request
		value, _ := h.requests.LoadOrStore(userID, &pendingRequest{})
		req := value.(*pendingRequest)

		// Check if callback is valid
		for _, cb := range req.callbacks {
			var unique string
			if c.Callback().Unique == "" {
				data := strings.Split(c.Callback().Data, "|")
				unique = strings.TrimSpace(data[0])
			} else {
				unique = strings.TrimSpace(c.Callback().Unique)
			}

			if strings.TrimSpace(cb.CallbackUnique()) == unique {
				_ = c.Respond(&tele.CallbackResponse{})
				// Set callback and mark as completed
				req.mu.Lock()
				req.callback = c.Callback()
				req.message = c.Message()
				req.completed = true
				req.mu.Unlock()

				// Clean up storage
				h.storage.Delete(userID)

				return nil
			}
		}

		return nil
	}
}

// Cancel cancels the input request for the given user
func (h *InputManager) Cancel(userID int64) {
	value, ok := h.requests.Load(userID)
	if !ok {
		return
	}

	req := value.(*pendingRequest)
	req.mu.Lock()
	req.canceled = true
	req.completed = true
	req.mu.Unlock()

	h.storage.Delete(userID)
	h.requests.Delete(userID)
}

// Response represents either a text message or a callback response
type Response struct {
	Message  *tele.Message
	Callback *tele.Callback
	Canceled bool
}

// Get waits for user input or callback and returns it. If timeout is 0, waits indefinitely.
// You can pass callback endpoints to handle button presses. If a button with matching unique identifier
// is pressed, the function will return a Response with Callback field set and Message that called the button.
//
// NOTE:
//   - This function is blocking, so make sure to call it in a separate goroutine
//   - It will return error if the context is canceled or context deadline is exceeded
//   - It will return ErrTimeout if the timeout is exceeded
//   - It will return nil error and Response.Canceled=true if input canceled by Cancel
//   - For text messages, Message field will be set and Callback will be nil
//   - For button callbacks, Message will be nil and Callback will contain the callback data
func (h *InputManager) Get(ctx context.Context, userID int64, timeout time.Duration, callback ...tele.CallbackEndpoint) (Response, error) {
	// Create request
	req := &pendingRequest{
		callbacks: callback,
	}
	h.requests.Store(userID, req)

	// Set the state
	if err := h.storage.Set(userID, stateWaitingInput, timeout); err != nil {
		h.requests.Delete(userID)
		return Response{}, err
	}

	// Clean up when we're done
	defer func() {
		h.storage.Delete(userID)
		h.requests.Delete(userID)
	}()

	// Wait for response with polling
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return Response{Canceled: true}, ctx.Err()
		default:
			req.mu.Lock()
			if req.completed {
				canceled := req.canceled
				response := Response{
					Message:  req.message,
					Callback: req.callback,
					Canceled: canceled,
				}
				req.mu.Unlock()
				return response, nil
			}
			req.mu.Unlock()

			if timeout > 0 && time.Since(start) > timeout {
				return Response{}, ErrTimeout
			}

			// Small sleep to prevent CPU spinning
			time.Sleep(100 * time.Millisecond)
		}
	}
}
