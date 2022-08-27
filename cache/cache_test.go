package cache

import (
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"os"
	"testing"
)

var redisCache *Cache
var devCache *CacheDev

type suite struct {
	name  string
	cache internal.Volatilizer
}

func TestMain(m *testing.M) {
	config.Current = config.LoadConfig()
	logz := logger.Get(config.Current)
	redisCache = NewCache(logz)
	devCache = NewDevCache(logz)
	os.Exit(m.Run())
}

func TestCacheSubscribe(t *testing.T) {
	tests := []suite{
		{name: "subscribe with redis cache", cache: redisCache},
		{name: "subscribe with dev mem cache", cache: devCache},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receiver := make(chan internal.Command)
			closeCn := make(chan bool)

			payload := internal.Command{Type: internal.MsgTypeError, Data: "invalid token", Channel: "random_cahn"}

			go tc.cache.Subscribe(receiver, "", payload.Channel, closeCn)

			err := tc.cache.Publish(payload)
			if err != nil {
				t.Error(err.Error())
			}
			res := <-receiver
			if res != payload {
				t.Error("Incorrect message is received")
			}
			close(receiver)
			closeCn <- true
			close(closeCn)
		})
	}
}
