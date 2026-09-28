package kafka

import (
	"context"
	"uuid"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

const headerEventID = "event-id"

type Publisher struct {
	cl *kgo.Client
}

func NewPublisher(cl *kgo.Client) *Publisher {
	return &Publisher{
		cl: cl,
	}
}

func (p *Publisher) Publish(ctx context.Context, msgs []outbox.Message) []outbox.Result {
	records := make([]*kgo.Record, 0, len(msgs))
	ids := make(map[*kgo.Record]uuid.UUID, len(msgs))

	for _, m := range msgs {
		rec := &kgo.Record{
			Topic: m.Topic,
			Key:   []byte(m.Key),
			Value: m.Payload,
			Headers: []kgo.RecordHeader{
				{
					Key:   headerEventID,
					Value: []byte(m.ID.String()),
				},
			},
		}

		records = append(records, rec)
		ids[rec] = m.ID
	}

	produced := p.cl.ProduceSync(ctx, records...)
	results := make([]outbox.Result, 0, len(produced))

	for _, res := range produced {
		results = append(
			results, outbox.Result{
				ID:  ids[res.Record],
				Err: res.Err,
			},
		)
	}

	return results
}
