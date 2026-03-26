package ds

import (
	"stormsim/pkg/model"
)

type Tasks[T any] struct {
	taskQueue *Queue[T]
	validTask map[model.EventType]struct{} //validates the task from the queue
}

func NewTasks[T any](validEventTypes []model.EventType) *Tasks[T] {
	validTask := make(map[model.EventType]struct{})
	for _, eventType := range validEventTypes {
		validTask[eventType] = struct{}{}
	}
	return &Tasks[T]{
		taskQueue: &Queue[T]{},
		validTask: validTask,
	}
}

// Send task to Queue
func (t *Tasks[T]) AssignTask(e T) {
	t.taskQueue.Enqueue(e)
}

// Pop Task
func (t *Tasks[T]) PopTask() (T, bool) {
	return t.taskQueue.Dequeue()
}

// Peek Task, not remove from Queue
func (t *Tasks[T]) PeekTask() (T, bool) {
	return t.taskQueue.Peek()
}

func (t *Tasks[T]) CheckValidTask(EventType *model.EventType) bool {
	if _, exist := t.validTask[*EventType]; !exist {
		t.taskQueue.Dequeue()
		return false
	}
	return true
}
