package pubsub


type Engine struct {
	Memory  *EventBus
	Storage *PersistentEventBus
}


func NewEngine() (*Engine, error) {
	persist, err := NewPersistent("data/pubsub.json")
	if err != nil {
		return nil, err
	}
	return &Engine{
		Memory:  NewInMem(),
		Storage: persist,
	}, nil
}