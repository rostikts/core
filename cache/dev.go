package cache

import (
	"encoding/json"
	"errors"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
)

type CacheDev struct {
	data     map[string]string
	log      *logger.Logger
	observer Observer
}

func NewDevCache(log *logger.Logger) *CacheDev {
	return &CacheDev{
		data:     make(map[string]string),
		observer: NewObserver(),
		log:      log,
	}
}

func (d *CacheDev) Get(key string) (val string, err error) {
	val, ok := d.data[key]
	if !ok {
		err = errors.New("key not found in cache")
	}
	return
}

func (d *CacheDev) Set(key string, value string) error {
	d.data[key] = value
	return nil
}

func (d *CacheDev) GetTyped(key string, v any) error {
	val, err := d.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), v)
}

func (d *CacheDev) SetTyped(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return d.Set(key, string(b))
}

func (d *CacheDev) Inc(key string, by int64) (n int64, err error) {
	if err = d.GetTyped(key, &n); err != nil {
		return
	}

	n += by

	err = d.SetTyped(key, n)
	return
}

func (d *CacheDev) Dec(key string, by int64) (int64, error) {
	return d.Inc(key, -1*by)
}

func (d *CacheDev) Subscribe(send chan internal.Command, token, channel string, close chan bool) {
	pubsub := d.observer.Subscribe(channel)

	ch := pubsub.Channel()

	for {
		select {
		case m := <-ch:
			var msg internal.Command
			if err := json.Unmarshal([]byte(m.(string)), &msg); err != nil {
				d.log.Error().Err(err).Msg("error parsing JSON message")
				_ = pubsub.Close()
				return
			}
			send <- msg
		case <-close:
			_ = pubsub.Close()
			return
		}
	}
}

func (d *CacheDev) Publish(msg internal.Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the event to system so server-side function can trigger
	go func(sysmsg internal.Command) {
		sysmsg.IsSystemEvent = true
		b, err := json.Marshal(sysmsg)
		if err != nil {
			d.log.Error().Err(err).Msg("error marshaling the system msg")
			return
		}
		d.observer.Publish("sbsys", string(b))
	}(msg)

	d.observer.Publish(msg.Channel, string(b))
	return nil
}

func (d *CacheDev) PublishDocument(channel, typ string, v any) {
	//TODO: Implement this
}

func (d *CacheDev) QueueWork(key, value string) error {
	var queue []string
	if err := d.GetTyped(key, &queue); err != nil {
		queue = make([]string, 0)
	}

	queue = append(queue, value)

	return d.SetTyped(key, queue)
}

func (d *CacheDev) DequeueWork(key string) (val string, err error) {
	var queue []string
	if err = d.GetTyped(key, &queue); err != nil {
		return
	} else if len(queue) == 0 {
		return
	}

	val = queue[0]

	err = d.SetTyped(key, queue[1:])
	return
}
