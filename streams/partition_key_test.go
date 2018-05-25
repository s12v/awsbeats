package streams

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"testing"
)

func TestXidPartitionKey(t *testing.T) {
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{"foo": "bar"}}}

	xidProvider := newXidPartitionKeyProvider()
	xidKey, err := xidProvider.PartitionKeyFor(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if xidKey == "" || xidKey == "bar" {
		t.Fatalf("uenxpected partition key: %s", xidKey)
	}
}
