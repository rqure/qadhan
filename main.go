package main

import (
	"os"

	qmq "github.com/rqure/qmq/src"
)

func main() {
	os.Setenv("QMQ_ADDR", "localhost:6379")
	os.Setenv("CITY", "Edmonton")
	os.Setenv("COuNTRY", "CA")
	engine := qmq.NewDefaultEngine(qmq.DefaultEngineConfig{
		NameProvider:               &NameProvider{},
		TransformerProviderFactory: &TransformerProviderFactory{},
		ProducerFactory:            &ProducerFactory{},
		EngineProcessor:            &EngineProcessor{},
	})
	engine.Run()
}
