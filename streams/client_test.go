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

func TestCreateXidPartitionKeyProvider(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	config := &StreamsConfig{PartitionKeyProvider: "xid"}
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}

	xidProvider := createPartitionKeyProvider(config)
	xidKey, err := xidProvider.PartitionKeyFor(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if xidKey == "" || xidKey == expectedPartitionKey {
		t.Fatalf("uenxpected partition key: %s", xidKey)
	}
}

func TestCreateFieldPartitionKeyProvider(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	config := &StreamsConfig{PartitionKey: fieldForPartitionKey}
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	fieldProvider := createPartitionKeyProvider(config)
	fieldKey, err := fieldProvider.PartitionKeyFor(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fieldKey != expectedPartitionKey {
		t.Fatalf("uenxpected partition key: %s", fieldKey)

	}
}

func TestMapEvent(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	provider := newFieldPartitionKeyProvider(fieldForPartitionKey)
	client := client{encoder: MockCodec{}, partitionKeyProvider: provider}
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
	provider := newFieldPartitionKeyProvider(fieldForPartitionKey)

	client := client{encoder: MockCodec{}, partitionKeyProvider: provider}
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	events := []publisher.Event{event}
	okEvents, records, _ := client.mapEvents(events)

	if len(records) != 1 {
		t.Errorf("Expected 1 records, got %v", len(records))
	}

	if len(okEvents) != 1 {
		t.Errorf("Expected 1 ok events, got %v", len(okEvents))
	}

	if string(records[0].Data) != "boom" {
		t.Errorf("Unexpected data %s", records[0].Data)
	}
}
