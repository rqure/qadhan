package main

import qmq "github.com/rqure/qmq/src"

type ProducerFactory struct{}

func (p *ProducerFactory) Create(key string, components qmq.EngineComponentProvider) qmq.Producer {
	maxLength := 10

	if key == "prayer:time:queue" {
		maxLength = 500
	}

	redisConnection := components.WithConnectionProvider().Get("redis").(*qmq.RedisConnection)
	transformerKey := "producer:" + key
	return qmq.NewRedisProducer(key, redisConnection, int64(maxLength), components.WithTransformerProvider().Get(transformerKey))
}
