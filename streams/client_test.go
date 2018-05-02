package streams

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"testing"
)

type MockCodec struct {
}

func (mock MockCodec) Encode(index string, event *beat.Event) ([]byte, error) {
	return []byte("boom"), nil
}

func TestMapEvent(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	client := client{encoder: MockCodec{}, partitionKey: fieldForPartitionKey}
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	record, err := client.mapEvent(event)

	if err != nil {
		t.Fatalf("uenxpected error: %v", err)
	}

	if string(record.Data) != "boom" {
		t.Errorf("Unexpected data: %s", record.Data)
	}

	actualPartitionKey := aws.StringValue(record.PartitionKey)
	if actualPartitionKey != expectedPartitionKey {
		t.Errorf("unexpected partition key: %s", actualPartitionKey)
	}
}

func TestMapEvents(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	client := client{encoder: MockCodec{}, partitionKey: fieldForPartitionKey}
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	events := []publisher.Event{event}
	records, _ := client.mapEvents(events)

	if len(records) != 1 {
		t.Errorf("Expected 1 records, got %v", len(records))
	}

	if string(records[0].Data) != "boom" {
		t.Errorf("Unexpected data %s", records[0].Data)
	}
}
