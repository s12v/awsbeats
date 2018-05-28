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
	streams              *kinesis.Kinesis
	streamName           string
	partitionKeyProvider PartitionKeyProvider
	beatName             string
	encoder              codec.Codec
	timeout              time.Duration
	observer             outputs.Observer
}

func newClient(sess *session.Session, config *StreamsConfig, observer outputs.Observer, beat beat.Info) (*client, error) {
	partitionKeyProvider := createPartitionKeyProvider(config)
	client := &client{
		streams:              kinesis.New(sess),
		streamName:           config.DeliveryStreamName,
		partitionKeyProvider: partitionKeyProvider,
		beatName:             beat.Beat,
		encoder:              json.New(false, beat.Version),
		timeout:              config.Timeout,
		observer:             observer,
	}

	return client, nil
}

func createPartitionKeyProvider(config *StreamsConfig) PartitionKeyProvider {
	if config.PartitionKeyProvider == "xid" {
		return newXidPartitionKeyProvider()
	} else {
		return newFieldPartitionKeyProvider(config.PartitionKey)
	}
}

func (client *client) Close() error {
	return nil
}

func (client *client) Connect() error {
	return nil
}

func (client *client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	rest, err := client.publishEvents(events)
	if len(rest) == 0 {
		// We have to ACK only when all the submission succeeded
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L232
		batch.ACK()
	} else {
		// Mark the failed events to retry
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L234
		batch.RetryEvents(rest)
	}
	return err
}

func (client *client) publishEvents(events []publisher.Event) ([]publisher.Event, error) {
	observer := client.observer
	observer.NewBatch(len(events))

	logp.Debug("kinesis", "received events: %v", events)
	okEvents, records, dropped := client.mapEvents(events)
	if dropped > 0 {
		logp.Debug("kinesis", "sent %d records: %v", len(records), records)
		observer.Dropped(dropped)
		observer.Acked(len(okEvents))
	}
	logp.Debug("kinesis", "mapped to records: %v", records)
	res, err := client.putKinesisRecords(records)
	if err != nil {
		if res == nil {
			logp.Critical("permanently failed to send %d records: %v", len(events), err)
			return []publisher.Event{}, nil
		}
		failed := collectFailedEvents(res, events)
		logp.Info("retrying %d events on error: %v", len(failed), err)
		return failed, err
	}
	return []publisher.Event{}, nil
}

func (client *client) mapEvents(events []publisher.Event) ([]publisher.Event, []*kinesis.PutRecordsRequestEntry, int) {
	dropped := 0
	records := make([]*kinesis.PutRecordsRequestEntry, 0, len(events))
	okEvents := make([]publisher.Event, 0, len(events))
	for i := range events {
		event := events[i]
		record, err := client.mapEvent(&event)
		if err != nil {
			logp.Debug("kinesis", "failed to map event(%v): %v", event, err)
			dropped++
		} else {
			okEvents = append(okEvents, event)
			records = append(records, record)
		}
	}
	return okEvents, records, dropped
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

	partitionKey, err := client.partitionKeyProvider.PartitionKeyFor(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get parititon key: %v", err)
	}

	return &kinesis.PutRecordsRequestEntry{Data: buf, PartitionKey: aws.String(partitionKey)}, nil
}
func (client *client) putKinesisRecords(records []*kinesis.PutRecordsRequestEntry) (*kinesis.PutRecordsOutput, error) {
	request := kinesis.PutRecordsInput{
		StreamName: &client.streamName,
		Records:    records,
	}
	res, err := client.streams.PutRecords(&request)
	if err != nil {
		return nil, fmt.Errorf("failed to put records: %v", err)
	}
	return res, nil
}

func collectFailedEvents(res *kinesis.PutRecordsOutput, events []publisher.Event) []publisher.Event {
	if *res.FailedRecordCount > 0 {
		failedEvents := make([]publisher.Event, 0)
		records := res.Records
		for i, r := range records {
			if r == nil {
				// See https://github.com/s12v/awsbeats/issues/27 for more info
				logp.Warn("no record returned from kinesis for event: ", events[i])
				continue
			}
			if *r.ErrorCode != "" {
				failedEvents = append(failedEvents, events[i])
			}
		}
		logp.Warn("Retrying %d events", len(failedEvents))
		return failedEvents
	}
	return []publisher.Event{}
}
