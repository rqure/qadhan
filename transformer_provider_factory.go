package main

import qmq "github.com/rqure/qmq/src"

type TransformerProviderFactory struct{}

func (t *TransformerProviderFactory) Create(components qmq.EngineComponentProvider) qmq.TransformerProvider {
	transformerProvider := qmq.NewDefaultTransformerProvider()
	transformerProvider.Set("producer:prayer:time:queue", []qmq.Transformer{
		qmq.NewProtoToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToMessageTransformer(components.WithLogger()),
	})
	transformerProvider.Set("producer:audio-player:file:exchange", []qmq.Transformer{
		qmq.NewProtoToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToMessageTransformer(components.WithLogger()),
	})
	transformerProvider.Set("consumer:prayer:time:queue", []qmq.Transformer{
		qmq.NewMessageToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToProtoTransformer(components.WithLogger(), &qmq.Prayer{}),
	})
	return transformerProvider
}
