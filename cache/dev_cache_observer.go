package cache

import (
	"errors"
	"sync"
)

type Observer interface {
	Subscribe(channel string) Subscriber
	Publish(channel string, msg interface{})
	PubNumSub(channel string) map[string]int
}

type Subscriber interface {
	Channel() <-chan interface{}
	Close() error
}

type memSubscriber struct {
	msgCh chan interface{}
}

func NewSubscriber() *memSubscriber {
	sub := &memSubscriber{}
	sub.msgCh = make(chan interface{}, 100)
	return sub
}

func (ps *memSubscriber) Channel() <-chan interface{} {
	return ps.msgCh
}

func (ps *memSubscriber) Close() error {
	_, ok := <-ps.msgCh
	if !ok {
		return errors.New("channel is already closed")
	}
	close(ps.msgCh)
	return nil
}

type memObserver struct {
	Subscriptions map[string][]*memSubscriber
	mx            sync.Mutex
}

func NewObserver() Observer {
	subs := make(map[string][]*memSubscriber)
	return &memObserver{Subscriptions: subs, mx: sync.Mutex{}}
}

func (o *memObserver) Subscribe(channel string) Subscriber {
	o.mx.Lock()
	defer o.mx.Unlock()

	newSub := NewSubscriber()
	if subs, ok := o.Subscriptions[channel]; ok {
		o.Subscriptions[channel] = append(subs, newSub)
	} else if !ok {
		o.Subscriptions[channel] = append([]*memSubscriber{}, newSub)
	}
	return newSub
}

func (o *memObserver) Publish(channel string, msg interface{}) {
	o.mx.Lock()
	defer o.mx.Unlock()

	for _, sub := range o.Subscriptions[channel] {
		sub.msgCh <- msg
	}
}

func (o *memObserver) PubNumSub(channel string) map[string]int {
	res := make(map[string]int)
	o.mx.Lock()
	defer o.mx.Unlock()
	res[channel] = len(o.Subscriptions[channel])
	return res
}
