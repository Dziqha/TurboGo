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

func (e *Engine) PublishAll(topic string, data []byte) error {
	if err := e.Storage.Publish(topic, data); err != nil {
		return err
	}
	if err := e.Memory.Publish(topic, data); err != nil {
		return err
	}
	return nil
}

func (p *Engine) SubscribeAll(topic string) <-chan []byte {
	memCh := p.Memory.Subscribe(topic)
	storageCh := p.Storage.Subscribe(topic)

	merged := make(chan []byte, 1000)

	go func() {
		for msg := range memCh {
			merged <- msg
		}
	}()
	go func() {
		for msg := range storageCh {
			merged <- msg
		}
	}()

	return merged
}
