package main

import qmq "github.com/rqure/qmq/src"

type TransformerProviderFactory struct{}

func (t *TransformerProviderFactory) Create(components qmq.EngineComponentProvider) qmq.TransformerProvider {
	transformerProvider := qmq.NewDefaultTransformerProvider()
	transformerProvider.Set("producer:prayer:times", []qmq.Transformer{
		qmq.NewProtoToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToMessageTransformer(components.WithLogger(), qmq.AnyToMessageTransformerConfig{
			SourceProvider: components.WithNameProvider(),
		}),
		qmq.NewTracePushTransformer(components.WithLogger()),
	})
	transformerProvider.Set("producer:audio-player:cmd:play-file", []qmq.Transformer{
		qmq.NewProtoToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToMessageTransformer(components.WithLogger(), qmq.AnyToMessageTransformerConfig{
			SourceProvider: components.WithNameProvider(),
		}),
		qmq.NewTracePushTransformer(components.WithLogger()),
	})
	transformerProvider.Set("producer:audio-player:cmd:play-tts", []qmq.Transformer{
		qmq.NewProtoToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToMessageTransformer(components.WithLogger(), qmq.AnyToMessageTransformerConfig{
			SourceProvider: components.WithNameProvider(),
		}),
		qmq.NewTracePushTransformer(components.WithLogger()),
	})
	transformerProvider.Set("consumer:prayer:times", []qmq.Transformer{
		qmq.NewTracePopTransformer(components.WithLogger()),
		qmq.NewMessageToAnyTransformer(components.WithLogger()),
		qmq.NewAnyToProtoTransformer(components.WithLogger(), &qmq.Prayer{}),
	})
	return transformerProvider
}
