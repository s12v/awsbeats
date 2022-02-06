package streams

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
)

type StubCodec struct {
	dat [][]byte
	err []error
}

func (c *StubCodec) Encode(index string, event *beat.Event) ([]byte, error) {
	dat, err := c.dat[0], c.err[0]
	c.dat = c.dat[1:]
	c.err = c.err[1:]
	return dat, err
}

type StubClient struct {
	calls []*kinesis.PutRecordsInput
	out   []*kinesis.PutRecordsOutput
	err   []error
}

func (c *StubClient) PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	c.calls = append(c.calls, input)
	out, err := c.out[0], c.err[0]
	c.out = c.out[1:]
	c.err = c.err[1:]
	return out, err
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
	codecData := [][]byte{[]byte("boom")}
	codecErr := []error{nil}
	client := &client{encoder: &StubCodec{dat: codecData, err: codecErr}, partitionKeyProvider: provider}
	event := &publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	_, record, err := client.mapEvent(event)

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

	codecData := [][]byte{[]byte("boom")}
	codecErr := []error{nil}
	client := client{encoder: &StubCodec{dat: codecData, err: codecErr}, partitionKeyProvider: provider}
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{fieldForPartitionKey: expectedPartitionKey}}}
	events := []publisher.Event{event}
	batches := client.mapEvents(events)
	okEvents, records, _ := batches[0].okEvents, batches[0].records, batches[0].dropped

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
		codecData := [][]byte{[]byte("boom")}
		codecErr := []error{nil}
		client.encoder = &StubCodec{dat: codecData, err: codecErr}

		putRecordsOut := []*kinesis.PutRecordsOutput{
			&kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String(""),
					},
				},
				FailedRecordCount: aws.Int64(0),
			},
		}
		putRecordsErr := []error{nil}
		client.streams = &StubClient{out: putRecordsOut, err: putRecordsErr}
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
		codecData := [][]byte{[]byte("")}
		codecErr := []error{fmt.Errorf("failed to encode")}
		client.encoder = &StubCodec{dat: codecData, err: codecErr}
		putRecordsOut := []*kinesis.PutRecordsOutput{
			&kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String(""),
					},
				},
				FailedRecordCount: aws.Int64(0),
			},
		}
		putRecordsErr := []error{nil}
		client.streams = &StubClient{out: putRecordsOut, err: putRecordsErr}

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
		codecData := [][]byte{[]byte("boom")}
		codecErr := []error{nil}
		client.encoder = &StubCodec{dat: codecData, err: codecErr}

		putRecordsOut := []*kinesis.PutRecordsOutput{
			&kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					nil,
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		putRecordsErr := []error{nil}
		client.streams = &StubClient{out: putRecordsOut, err: putRecordsErr}

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
		codecData := [][]byte{[]byte("boom")}
		codecErr := []error{nil}
		client.encoder = &StubCodec{dat: codecData, err: codecErr}

		putRecordsOut := []*kinesis.PutRecordsOutput{
			&kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: nil,
					},
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		putRecordsErr := []error{nil}
		client.streams = &StubClient{out: putRecordsOut, err: putRecordsErr}

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
		codecData := [][]byte{[]byte("boom")}
		codecErr := []error{nil}
		client.encoder = &StubCodec{dat: codecData, err: codecErr}

		putRecordsOut := []*kinesis.PutRecordsOutput{
			&kinesis.PutRecordsOutput{
				Records: []*kinesis.PutRecordsResultEntry{
					&kinesis.PutRecordsResultEntry{
						ErrorCode: aws.String("simulated_error"),
					},
				},
				FailedRecordCount: aws.Int64(1),
			},
		}
		putRecordsErr := []error{nil}
		client.streams = &StubClient{out: putRecordsOut, err: putRecordsErr}

		rest, err := client.publishEvents(events)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(rest) != 1 {
			t.Errorf("unexpected number of remaining events: %d", len(rest))
		}
	}
}

func TestTestPublishEventsBatch(t *testing.T) {
	events := []publisher.Event{}
	fieldForPartitionKey := "mypartitionkey"
	provider := newFieldPartitionKeyProvider(fieldForPartitionKey)
	client := client{
		partitionKeyProvider: provider,
		observer:             outputs.NewNilObserver(),
	}
	codecData := [][]byte{
		[]byte(strings.Repeat("a", 500000)),
		[]byte(strings.Repeat("a", 500000)),
		[]byte(strings.Repeat("a", 500000)),
		[]byte(strings.Repeat("a", 900000)),
		[]byte(strings.Repeat("a", 900000)),
		[]byte(strings.Repeat("a", 900000)),
		[]byte(strings.Repeat("a", 900000)),
		[]byte(strings.Repeat("a", 900000)),
		[]byte(strings.Repeat("a", 900000)),
	}
	codecErr := []error{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}
	client.encoder = &StubCodec{dat: codecData, err: codecErr}

	putRecordsOutputGood := &kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{
			&kinesis.PutRecordsResultEntry{
				ErrorCode: aws.String(""),
			},
		},
		FailedRecordCount: aws.Int64(0),
	}
	putRecordsOut := []*kinesis.PutRecordsOutput{
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
		putRecordsOutputGood,
	}
	putRecordsErr := []error{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	}
	kinesisStub := &StubClient{out: putRecordsOut, err: putRecordsErr}
	client.streams = kinesisStub

	for _, _ = range putRecordsErr {
		events = append(events, publisher.Event{
			Content: beat.Event{
				Fields: common.MapStr{
					fieldForPartitionKey: "expectedPartitionKey",
				},
			},
		},
		)
	}
	rest, err := client.publishEvents(events)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(rest) != 0 {
		t.Errorf("unexpected number of remaining events: %d", len(rest))
	}
	if len(kinesisStub.calls) != 2 {
		t.Errorf("unexpected number of batches: %d", len(kinesisStub.calls))
	}
	if len(kinesisStub.calls[0].Records) != 6 {
		t.Errorf("unexpected number of events in batch 0 batches: %d", len(kinesisStub.calls[0].Records))
	}
	if len(kinesisStub.calls[1].Records) != 3 {
		t.Errorf("unexpected number of events in batch 1 batches: %d", len(kinesisStub.calls[1].Records))
	}
}

func TestClient_String(t *testing.T) {
	fieldForPartitionKey := "mypartitionkey"
	provider := newFieldPartitionKeyProvider(fieldForPartitionKey)
	codecData := [][]byte{[]byte("boom")}
	codecErr := []error{nil}
	client := client{encoder: &StubCodec{dat: codecData, err: codecErr}, partitionKeyProvider: provider}

	if v := client.String(); v != "streams" {
		t.Errorf("unexpected value '%v'", v)
	}
}
