package pubsub

import (
	"errors"
	"sync"
)

type EventBus struct {
	mu     sync.RWMutex
	topics map[string][]chan []byte
	closed bool
}

func NewInMem() *EventBus {
	return &EventBus{
		topics: make(map[string][]chan []byte),
		closed: false,
	}
}

func (b *EventBus) Publish(topic string, msg []byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if b.closed {
		return errors.New("eventbus is closed")
	}
	
	channels := b.topics[topic]
	for _, ch := range channels {
		select {
		case ch <- msg:
		default:
			// Channel is full, skip
		}
	}
	return nil
}

func (b *EventBus) Subscribe(topic string) <-chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.closed {
		return nil
	}
	
	ch := make(chan []byte, 10000)
	b.topics[topic] = append(b.topics[topic], ch)
	return ch
}

func (b *EventBus) SubscribeRaw(topic string) (<-chan []byte, error) {
	if b.closed {
		return nil, errors.New("eventbus is closed")
	}

	ch := make(chan []byte, 10000)

	b.mu.Lock()
	b.topics[topic] = append(b.topics[topic], ch)
	b.mu.Unlock()

	return ch, nil
}


func (b *EventBus) Unsubscribe(topic string, ch <-chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	channels := b.topics[topic]
	for i, subscriber := range channels {
		if subscriber == ch {
			b.topics[topic] = append(channels[:i], channels[i+1:]...)
			close(subscriber)
			break
		}
	}
}

func (b *EventBus) UnsubscribeAll(topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	chs := b.topics[topic]
	for _, ch := range chs {
		close(ch)
	}
	delete(b.topics, topic)
}


func (b *EventBus) GetTopics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	topics := make([]string, 0, len(b.topics))
	for topic := range b.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (b *EventBus) GetSubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.topics[topic])
}

func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.closed = true
	for topic, channels := range b.topics {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.topics, topic)
	}
}

