package kafka

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kslog"
)

type ProducerOptions struct {
	Brokers         []string
	ClientID        string
	DeliveryTimeout time.Duration
	Log             *slog.Logger
}

func NewProducerClient(opts ProducerOptions) (*kgo.Client, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(opts.Brokers...),
		kgo.ClientID(opts.ClientID),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(opts.DeliveryTimeout),
		kgo.AllowIdempotentProduceCancellation(),
		kgo.ProducerBatchCompression(kgo.ZstdCompression(), kgo.SnappyCompression(), kgo.NoCompression()),
		kgo.WithLogger(kslog.New(opts.Log)),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka producer client: %w", err)
	}

	return client, nil
}
