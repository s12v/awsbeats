package streams

import (
	"fmt"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/rs/xid"
)

type PartitionKeyProvider interface {
	PartitionKeyFor(event *publisher.Event) (string, error)
}

type fieldPartitionKeyProvider struct {
	fieldKey string
}

type xidPartitionKeyProvider struct {
}

func newFieldPartitionKeyProvider(fieldKey string) *fieldPartitionKeyProvider {
	return &fieldPartitionKeyProvider{
		fieldKey: fieldKey,
	}
}

func (p *fieldPartitionKeyProvider) PartitionKeyFor(event *publisher.Event) (string, error) {
	rawPartitionKey, err := event.Content.GetValue(p.fieldKey)
	if err != nil {
		return "", fmt.Errorf("failed to get parition key: %v", err)
	}

	var ok bool
	partitionKey, ok := rawPartitionKey.(string)
	if !ok {
		return "", fmt.Errorf("failed to get partition key: %s(=%v) is found, but not a string", p.fieldKey, rawPartitionKey)
	}

	return partitionKey, nil
}

func newXidPartitionKeyProvider() *xidPartitionKeyProvider {
	return &xidPartitionKeyProvider{}
}

func (p *xidPartitionKeyProvider) PartitionKeyFor(_ *publisher.Event) (string, error) {
	return xid.New().String(), nil
}
