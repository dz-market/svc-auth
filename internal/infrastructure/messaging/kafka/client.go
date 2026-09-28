package kafka

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kslog"
)

type Options struct {
	Brokers         []string
	ClientID        string
	DeliveryTimeout time.Duration
	Log             *slog.Logger
}

func NewClient(opts Options) (*kgo.Client, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(opts.Brokers...),
		kgo.ClientID(opts.ClientID),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(opts.DeliveryTimeout),
		kgo.AllowIdempotentProduceCancellation(),
		kgo.ProducerBatchCompression(kgo.ZstdCompression(), kgo.SnappyCompression(), kgo.NoCompression()),
		kgo.WithLogger(kslog.New(opts.Log)),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka client: %w", err)
	}

	return cl, nil
}
