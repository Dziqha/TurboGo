package queue


type Engine struct {
	Memory *TaskQueue
	Storage *PersistentTaskQueue
}



func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/queue.json")
	if err != nil {
		return nil, err
	}
	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}


func (e *Engine) EnqueueAll(queue string, task []byte) error {
	if err := e.Storage.Enqueue(queue, task); err != nil {
		return err
	}
	if err := e.Memory.Enqueue(queue, task); err != nil {
		return err
	}
	return nil
}


func (q *Engine) RegisterWorkerAll(queueName string, handler func([]byte) error) {
	q.Memory.RegisterWorker(queueName, handler)
	q.Storage.RegisterWorker(queueName, handler)
}
