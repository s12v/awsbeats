package streams

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
	"time"
)

type client struct {
	streams      *kinesis.Kinesis
	streamName   string
	partitionKey string
	beatName     string
	encoder      codec.Codec
	timeout      time.Duration
	observer     outputs.Observer
}

func newClient(sess *session.Session, config *StreamsConfig, observer outputs.Observer, beat beat.Info) (*client, error) {
	client := &client{
		streams:      kinesis.New(sess),
		streamName:   config.DeliveryStreamName,
		partitionKey: config.PartitionKey,
		beatName:     beat.Beat,
		encoder:      json.New(false, beat.Version),
		timeout:      config.Timeout,
		observer:     observer,
	}

	return client, nil
}

func (client *client) Close() error {
	return nil
}

func (client *client) Connect() error {
	return nil
}

func (client *client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	observer := client.observer
	observer.NewBatch(len(events))

	logp.Debug("kinesis", "received events: %v", events)
	records, dropped := client.mapEvents(events)
	logp.Debug("kinesis", "mapped to records: %v", records)
	res, err := client.sendRecords(records)
	if err != nil {
		logp.Critical("Unable to send batch: %v", err)
		observer.Dropped(len(events))
		return err
	}
	processFailedDeliveries(res, batch)

	batch.ACK()
	logp.Debug("kinesis", "sent %d records: %v", len(records), records)
	observer.Dropped(dropped)
	observer.Acked(len(events) - dropped)
	return nil
}

func (client *client) mapEvents(events []publisher.Event) ([]*kinesis.PutRecordsRequestEntry, int) {
	dropped := 0
	records := make([]*kinesis.PutRecordsRequestEntry, 0, len(events))
	for i := range events {
		event := events[i]
		record, err := client.mapEvent(&event)
		if err != nil {
			logp.Debug("kinesis", "failed to map event(%v): %v", event, err)
			dropped++
		} else {
			records = append(records, record)
		}
	}

	return records, dropped
}

func (client *client) mapEvent(event *publisher.Event) (*kinesis.PutRecordsRequestEntry, error) {
	var buf []byte
	{
		serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
		if err != nil {
			logp.Critical("Unable to encode event: %v", err)
			return nil, err
		}
		// See https://github.com/elastic/beats/blob/5a6630a8bc9b9caf312978f57d1d9193bdab1ac7/libbeat/outputs/kafka/client.go#L163-L164
		// You need to copy the byte data like this. Otherwise you see strange issues like all the records sent in a same batch has the same Data.
		buf = make([]byte, len(serializedEvent))
		copy(buf, serializedEvent)
	}

	rawPartitionKey, err := event.Content.GetValue(client.partitionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get parition key: %v", err)
	}

	partitionKey, ok := rawPartitionKey.(string)
	if !ok {
		return nil, fmt.Errorf("failed to get partition key: %s(=%v) is found, but not a string", client.partitionKey, rawPartitionKey)
	}

	return &kinesis.PutRecordsRequestEntry{Data: buf, PartitionKey: aws.String(partitionKey)}, nil
}
func (client *client) sendRecords(records []*kinesis.PutRecordsRequestEntry) (*kinesis.PutRecordsOutput, error) {
	request := kinesis.PutRecordsInput{
		StreamName: &client.streamName,
		Records:    records,
	}
	return client.streams.PutRecords(&request)
}

func processFailedDeliveries(res *kinesis.PutRecordsOutput, batch publisher.Batch) {
	if *res.FailedRecordCount > 0 {
		events := batch.Events()
		failedEvents := make([]publisher.Event, 0)
		records := res.Records
		for i, r := range records {
			if *r.ErrorCode != "" {
				failedEvents = append(failedEvents, events[i])
			}
		}

		if len(failedEvents) > 0 {
			logp.Warn("Retrying %d events", len(failedEvents))
			batch.RetryEvents(failedEvents)
			return
		}
	}
}
