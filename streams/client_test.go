package streams

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"testing"
)

type StubCodec struct {
	dat []byte
	err error
}

func (c StubCodec) Encode(index string, event *beat.Event) ([]byte, error) {
	return c.dat, c.err
}

type StubClient struct {
	out *kinesis.PutRecordsOutput
	err error
}

func (c StubClient) PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	return c.out, c.err
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
	client := client{encoder: StubCodec{dat: []byte("boom")}, partitionKeyProvider: provider}
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	record, err := client.mapEvent(event)

	if err != nil {
		t.Fatalf("uenxpected error: %v", err)
	}

	if string(record.Data) != "boom\n" {
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

	client := client{encoder: StubCodec{dat: []byte("boom")}, partitionKeyProvider: provider}
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	events := []publisher.Event{event}
	okEvents, records, _ := client.mapEvents(events)

	if len(records) != 1 {
		t.Errorf("Expected 1 records, got %v", len(records))
	}

	if len(okEvents) != 1 {
		t.Errorf("Expected 1 ok events, got %v", len(okEvents))
	}

	if string(records[0].Data) != "boom\n" {
		t.Errorf("Unexpected data %s", records[0].Data)
	}
}

func TestPublishEvents(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	expectedPartitionKey := "foobar"
	provider := newFieldPartitionKeyProvider(fieldForPartitionKey)
	client := client{
		partitionKeyProvider: provider,
		observer:             outputs.NewNilObserver(),
	}
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	events := []publisher.Event{event}

	{
		client.encoder = StubCodec{dat: []byte("boom"), err: nil}
		client.streams = StubClient{
			out: &kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String(""),
					},
				},
				FailedRecordCount: aws.Int64(0),
			},
		}
		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 0 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}

	{
		// An event that can't be encoded should be ignored without any error, but with some log.
		client.encoder = StubCodec{dat: []byte(""), err: fmt.Errorf("failed to encode")}
		client.streams = StubClient{
			out: &kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String(""),
					},
				},
				FailedRecordCount: aws.Int64(0),
			},
		}
		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 0 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}

	{
		// Nil records returned by Kinesis should be ignored with some log
		client.encoder = StubCodec{dat: []byte("boom"), err: nil}
		client.streams = StubClient{
			out: &kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					nil,
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 0 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}

	{
		// Records with nil error codes should be ignored with some log
		client.encoder = StubCodec{dat: []byte("boom"), err: nil}
		client.streams = StubClient{
			out: &kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: nil,
					},
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 0 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}

	{
		// Kinesis received the event but it was not persisted, probably due to underlying infrastructure failure
		client.encoder = StubCodec{dat: []byte("boom"), err: nil}
		client.streams = StubClient{
			out: &kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String("simulated_error"),
					},
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 1 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}
}
