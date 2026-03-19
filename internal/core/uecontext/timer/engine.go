package timer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TimerCallback represents a callback function for timeout or expiration
type TimerCallback func()

// TimerConfig holds the configuration for a timer
type TimerConfig struct {
	TimerType   TimerType     // Unique identifier for the timer
	Duration    time.Duration // Duration before timeout
	CountMax    int           // Maximum number of retries
	TimeoutFunc TimerCallback // Function to call on each timeout (can be nil)
	ExpireFunc  TimerCallback // Function to call when timer expires
}

// Timer represents a single timer with its state
type Timer struct {
	config        TimerConfig
	currentCount  int
	isActive      bool
	cancel        context.CancelFunc
	manualTrigger chan struct{}
	mu            sync.RWMutex
}

// TimerEngine manages multiple concurrent timers
type TimerEngine struct {
	timers          map[TimerType]*Timer
	BlockTimerEvent bool
	mu              sync.RWMutex
}

// NewTimerEngine creates a new timer engine
func NewTimerEngine() *TimerEngine {
	return &TimerEngine{
		timers: make(map[TimerType]*Timer),
	}
}

// CreateTimer creates a new timer with the given configuration
func (te *TimerEngine) CreateTimer(config TimerConfig) {
	te.mu.Lock()
	defer te.mu.Unlock()

	timer := &Timer{
		config:        config,
		currentCount:  0,
		isActive:      false,
		manualTrigger: make(chan struct{}, 1), // Buffered channel to prevent blocking
	}

	te.timers[config.TimerType] = timer
}

// Start begins the timer countdown in a goroutine
func (te *TimerEngine) Start(timerID TimerType) error {
	te.mu.RLock()
	timer, exists := te.timers[timerID]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer not found")
	}

	timer.mu.Lock()
	defer timer.mu.Unlock()

	if timer.isActive {
		return fmt.Errorf("timer is already active")
	}

	ctx, cancel := context.WithCancel(context.Background())
	timer.cancel = cancel
	timer.isActive = true
	timer.currentCount = 0

	// Start the timer in a goroutine
	go timer.run(ctx)

	return nil
}

// Stop stops the timer
func (te *TimerEngine) Stop(timerID TimerType) error {
	te.mu.RLock()
	timer, exists := te.timers[timerID]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer %v not found", timerID)
	}

	timer.mu.Lock()
	defer timer.mu.Unlock()

	if !timer.isActive {
		return fmt.Errorf("timer %v is not active", timerID)
	}

	timer.cancel()
	timer.isActive = false

	te.RemoveTimer(timerID) // rm timer
	return nil
}

// Stop stops the timer then start expire func()
func (te *TimerEngine) StopWithExpireFunc(timerID TimerType) error {
	te.mu.RLock()
	timer, exists := te.timers[timerID]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer %v not found", timerID)
	}

	timer.mu.Lock()
	defer timer.mu.Unlock()

	if !timer.isActive {
		return fmt.Errorf("timer %v is not active", timerID)
	}
	timer.cancel()
	timer.isActive = false

	timer.config.ExpireFunc() // start expire func()
	te.RemoveTimer(timerID)   // rm timer
	return nil
}

// TriggerTimeout manually triggers a timeout event for the specified timer
func (te *TimerEngine) TriggerTimeout(timerID TimerType) error {
	te.mu.RLock()
	timer, exists := te.timers[timerID]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer with ID %v not found", timerID)
	}

	timer.mu.RLock()
	isActive := timer.isActive
	timer.mu.RUnlock()

	if !isActive {
		return fmt.Errorf("timer %v is not active", timerID)
	}

	// Send manual trigger (non-blocking)
	select {
	case timer.manualTrigger <- struct{}{}:
		return nil
	default:
		// Channel is full, trigger already pending
		return fmt.Errorf("timer %v already has a pending manual trigger", timerID)
	}
}

// GetTimerStatus returns the current status of a timer
func (te *TimerEngine) GetTimerStatus(timerID TimerType) (bool, int, error) {
	te.mu.RLock()
	timer, exists := te.timers[timerID]
	te.mu.RUnlock()

	if !exists {
		return false, 0, nil
	}

	timer.mu.RLock()
	defer timer.mu.RUnlock()

	return timer.isActive, timer.currentCount, nil
}

// RemoveTimer removes a timer from the engine
func (te *TimerEngine) RemoveTimer(timerID TimerType) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	timer, exists := te.timers[timerID]
	if !exists {
		return fmt.Errorf("timer %v not found", timerID)
	}

	// Stop the timer if it's active
	timer.mu.Lock()
	if timer.isActive {
		timer.cancel()
		timer.isActive = false
	}
	timer.mu.Unlock()

	delete(te.timers, timerID)
	return nil
}

// ListActiveTimers returns a list of active timer IDs
func (te *TimerEngine) ListActiveTimers() []TimerType {
	te.mu.RLock()
	defer te.mu.RUnlock()

	var activeTimers []TimerType
	for id, timer := range te.timers {
		timer.mu.RLock()
		if timer.isActive {
			activeTimers = append(activeTimers, id)
		}
		timer.mu.RUnlock()
	}
	return activeTimers
}

// run executes the timer logic in a goroutine
func (t *Timer) run(ctx context.Context) {
	ticker := time.NewTicker(t.config.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Call expire function if provided
			if t.config.ExpireFunc != nil {
				go t.config.ExpireFunc()
			}
			return
		case <-t.manualTrigger:
			t.excute()
			ticker.Reset(t.config.Duration)
		case <-ticker.C:
			t.excute()
		}
		if t.currentCount >= t.config.CountMax {
			return
		}
	}
}

func (t *Timer) excute() {
	t.mu.Lock()
	t.currentCount++
	currentCount := t.currentCount
	t.mu.Unlock()

	// Check if we've reached the maximum count
	if currentCount == t.config.CountMax {
		// Timer has expired
		t.mu.Lock()
		t.isActive = false
		t.mu.Unlock()

		// Call expire function if provided
		if t.config.ExpireFunc != nil {
			go t.config.ExpireFunc()
		}
		return
	}

	// Call timeout function if provided
	if t.config.TimeoutFunc != nil {
		go t.config.TimeoutFunc()
	}
}
