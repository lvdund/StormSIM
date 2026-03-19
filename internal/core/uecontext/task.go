package uecontext

import (
	"slices"
	"stormsim/internal/common/ds"
	"stormsim/internal/common/fsm"
	"stormsim/pkg/model"
	"sync"
	"time"
)

const (
	TaskPollInterval   = 100 * time.Millisecond
	TaskTimeout        = time.Minute
	StateCheckInterval = 10 * time.Millisecond
)

type TaskType string

const (
	TaskTypeMM TaskType = "MM" // Mobility Management
	TaskTypeSM TaskType = "SM" // Session Management
)

type TaskStatus string

const (
	TaskStatusStarted   TaskStatus = "started"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusTimeout   TaskStatus = "timeout"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// taskInfo contains timing and metadata for task execution
type taskInfo struct {
	eventType    model.EventType
	taskType     TaskType
	targetState  model.StateType
	pduSessionId *uint8 // nil for MM tasks, set for SM tasks

	status TaskStatus
	// stat   *logger.TaskStat
}

// manages timing for active tasks
type taskTimer struct {
	mu          sync.RWMutex
	activeTasks map[model.EventType]*taskInfo
}

func (tt *taskTimer) taskAction(info *taskInfo, eventType *model.EventType, action TaskStatus) *taskInfo {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	switch action {
	case TaskStatusStarted:
		// info.stat.Start = time.Now()
		info.status = action
		tt.activeTasks[info.eventType] = info
	case TaskStatusCompleted, TaskStatusTimeout, TaskStatusCancelled:
		if info == nil {
			return nil
		}
		if info, exists := tt.activeTasks[*eventType]; exists {
			// endTime := time.Now()
			// duration := endTime.Sub(info.stat.Start)
			// info.stat.End = &endTime
			// info.stat.Durarion = &duration
			info.status = action

			delete(tt.activeTasks, *eventType)
			return info
		}
	}

	// if action == TaskStatusCompleted {
	// 	info.stat.Task += " -> done!"
	// }

	return nil
}

// taskTracker manages event-to-state mappings for both MM and SM state machines
type taskTracker struct {
	mmTracker map[model.EventType]model.StateType
	smTracker map[model.EventType]model.StateType

	taskTimer *taskTimer
}

func newTaskTracker() *taskTracker {
	return &taskTracker{
		mmTracker: map[model.EventType]model.StateType{
			// Registration events -> Registered state
			model.RegisterInit:                 model.Registered,
			model.InitRegistrationRequestEvent: model.Registered,
			model.ServiceRequestInit:           model.Registered,

			// Deregistration events -> Deregistered state
			model.DeregistraterInit:                 model.Deregistered,
			model.InitDeregistrationRequestEvent:    model.Deregistered,
			model.NetworkDeregistrationRequestEvent: model.Deregistered,
		},
		smTracker: map[model.EventType]model.StateType{
			// PDU session establishment -> Active state
			model.PduSessionInit:                          model.PDUSessionActive,
			model.InitPduSessionEstablishmentRequestEvent: model.PDUSessionActive,

			// PDU session release -> Inactive state
			model.DestroyPduSession: model.PDUSessionInactive,
			model.ReleaseRequest:    model.PDUSessionInactive,
			model.ReleaseCommand:    model.PDUSessionInactive,
		},
		taskTimer: &taskTimer{
			activeTasks: make(map[model.EventType]*taskInfo),
		},
	}
}

func (ue *UeContext) newTasks() {
	ue.eventQueue = ds.NewTasks[*EventUeData]([]model.EventType{
		// 5gmm
		model.RegisterInit, model.InitRegistrationRequestEvent,
		model.DeregistraterInit, model.Terminate,
		// 5gsm
		model.PduSessionInit, model.DestroyPduSession,
	})

	// Initialize task tracker and timer
	ue.taskTracker = newTaskTracker()

	ue.wg.Add(1)
	go ue.doTasks()
}

func (ue *UeContext) doTasks() {
	defer ue.wg.Done()

	for {
		select {
		case <-ue.ctx.Done():
			return
		default:
		}

		task, ok := ue.eventQueue.PeekTask()
		if !ok {
			time.Sleep(10 * time.Millisecond) // Avoid busy-wait
			continue
		}

		if !ue.eventQueue.CheckValidTask(&task.EventType) {
			ue.Error("[Task]: event %s is not supported", task.EventType)
			continue
		}

		ue.executeTaskWithTiming(task)
		ue.eventQueue.PopTask()
	}
}

// executeTaskWithTiming executes a task and handles timing/state tracking
func (ue *UeContext) executeTaskWithTiming(task *EventUeData) {
	taskInfo := ue.createTaskInfo(task.EventType)
	//logger.TaskStats[ue.msin] = append(logger.TaskStats[ue.msin], taskInfo.stat)

	if taskInfo != nil {
		ue.taskTracker.taskTimer.taskAction(taskInfo, nil, "start")
		ue.Info("[Task %s]: event %s: started, target state %s",
			taskInfo.taskType, task.EventType, taskInfo.targetState)
	} else {
		ue.Info("[Task]: event %s: execute (no state tracking)", task.EventType)
	}

	time.Sleep(time.Duration(task.Delay) * time.Second)
	// Execute the task: send task (event) to fsm
	ue.sendEventMm(fsm.NewEventData(task.EventType, &task.Params))

	// Wait for target state if task has tracking
	if taskInfo != nil {
		timeout := time.After(TaskTimeout)
		ticker := time.NewTicker(StateCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ue.ctx.Done():
				if info := ue.taskTracker.taskTimer.taskAction(taskInfo, &task.EventType, TaskStatusCancelled); info != nil {
					ue.Warn("[Task: %s]: event %s: cancel after %v, target state %s",
						taskInfo.taskType, task.EventType, TaskTimeout, taskInfo.targetState)
				}
				return

			case <-timeout:
				if info := ue.taskTracker.taskTimer.taskAction(taskInfo, &task.EventType, TaskStatusTimeout); info != nil {
					ue.Warn("[Task: %s]: event %s: timeout after %v, target state %s",
						taskInfo.taskType, task.EventType, TaskTimeout, taskInfo.targetState)
				}
				return

			case <-ticker.C:
				if ue.isTargetStateReached(task.EventType, taskInfo.taskType, taskInfo.targetState) {
					if info := ue.taskTracker.taskTimer.taskAction(taskInfo, &task.EventType, TaskStatusCompleted); info != nil {
						ue.Info("[Task: %s]: event %s: completed, reached state %s",
							taskInfo.taskType, task.EventType, taskInfo.targetState)
					}

					return
				}
			}
		}
	}
}

func (ue *UeContext) createTaskInfo(eventType model.EventType) *taskInfo {
	var targetState model.StateType
	var taskType TaskType
	var found bool

	// Check if it's an MM task
	if targetState, found = ue.taskTracker.mmTracker[eventType]; found {
		taskType = TaskTypeMM
	} else if targetState, found = ue.taskTracker.smTracker[eventType]; found {
		taskType = TaskTypeSM
	} else {
		return nil // Event not tracked
	}

	return &taskInfo{
		eventType:   eventType,
		taskType:    taskType,
		targetState: targetState,
		// stat: &logger.TaskStat{
		// 	Task: fmt.Sprintf("Task %s", eventType),
		// },
	}
}

func (ue *UeContext) isTargetStateReached(eventType model.EventType, taskType TaskType, targetState model.StateType) bool {
	switch taskType {
	case TaskTypeMM:
		return ue.state_mm.CurrentState() == targetState

	case TaskTypeSM: // Check if any PDU session has reached the target state

		for i := uint8(1); i < 16; i++ { // Check existing sessions for target state
			if pduSession := ue.sessions[i]; pduSession != nil {
				if pduSession.state_sm.CurrentState() == targetState {
					return true
				}
			}
		}

		// Special case: for destroy/release events, also consider it complete if no sessions exist
		if slices.Contains([]model.EventType{
			model.DestroyPduSession,
			model.ReleaseRequest,
			model.ReleaseCommand,
		}, eventType) {
			count := 0
			for i := uint8(1); i < 16; i++ {
				if ue.sessions[i] != nil {
					count++
				}
			}
			return count == 0
		}

		return false

	default:
		return false
	}
}
