package firehose

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
	"time"
)

type client struct {
	firehose           *firehose.Firehose
	deliveryStreamName string
	beatName           string
	encoder            codec.Codec
	timeout            time.Duration
	observer           outputs.Observer
}

func newClient(sess *session.Session, config *FirehoseConfig, observer outputs.Observer, beat beat.Info) (*client, error) {
	client := &client{
		firehose:           firehose.New(sess),
		deliveryStreamName: config.DeliveryStreamName,
		beatName:           beat.Beat,
		encoder:            json.New(false, true, beat.Version),
		timeout:            config.Timeout,
		observer:           observer,
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
	rest, _ := client.publishEvents(events)
	if len(rest) == 0 {
		// We have to ACK only when all the submission succeeded
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L232
		batch.ACK()
	} else {
		// Mark the failed events to retry
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L234
		batch.RetryEvents(rest)
	}
	// This shouldn't be an error object according to other official beats' implementations
	// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/kafka/client.go#L119
	return nil
}

func (client *client) publishEvents(events []publisher.Event) ([]publisher.Event, error) {
	observer := client.observer
	observer.NewBatch(len(events))

	logp.Debug("firehose", "received events: %v", events)
	okEvents, records, dropped := client.mapEvents(events)

	logp.Debug("firehose", "sent %d records: %v", len(records), records)
	observer.Dropped(dropped)
	observer.Acked(len(okEvents))

	logp.Debug("firehose", "mapped to records: %v", records)
	res, err := client.sendRecords(records)
	failed := collectFailedEvents(res, events)
	if err != nil && len(failed) == 0 {
		failed = events
	}
	if len(failed) > 0 {
		logp.Info("retrying %d events on error: %v", len(failed), err)
	}
	return failed, err
}

func (client *client) mapEvents(events []publisher.Event) ([]publisher.Event, []*firehose.Record, int) {
	dropped := 0
	records := make([]*firehose.Record, 0, len(events))
	okEvents := make([]publisher.Event, 0, len(events))
	for _, event := range events {
		record, err := client.mapEvent(&event)
		if err != nil {
			logp.Debug("firehose", "failed to map event(%v): %v", event, err)
			dropped++
		} else {
			okEvents = append(okEvents, event)
			records = append(records, record)
		}
	}

	return okEvents, records, dropped
}

func (client *client) mapEvent(event *publisher.Event) (*firehose.Record, error) {
	var buf []byte
	{
		serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
		if err != nil {
			if !event.Guaranteed() {
				return nil, err
			}

			logp.Critical("Unable to encode event: %v", err)
			return nil, err
		}
		// See https://github.com/elastic/beats/blob/5a6630a8bc9b9caf312978f57d1d9193bdab1ac7/libbeat/outputs/kafka/client.go#L163-L164
		// You need to copy the byte data like this. Otherwise you see strange issues like all the records sent in a same batch has the same Data.
		buf = make([]byte, len(serializedEvent)+1)
		copy(buf, serializedEvent)
		// Firehose doesn't automatically add trailing new-line on after each record.
		// This ends up a stream->firehose->s3 pipeline to produce useless s3 objects.
		// No ndjson, but a sequence of json objects without separators...
		// Fix it just adding a new-line.
		//
		// See https://stackoverflow.com/questions/43010117/writing-properly-formatted-json-to-s3-to-load-in-athena-redshift
		buf[len(buf)-1] = byte('\n')
	}

	return &firehose.Record{Data: buf}, nil
}
func (client *client) sendRecords(records []*firehose.Record) (*firehose.PutRecordBatchOutput, error) {
	request := firehose.PutRecordBatchInput{
		DeliveryStreamName: &client.deliveryStreamName,
		Records:            records,
	}
	return client.firehose.PutRecordBatch(&request)
}

func collectFailedEvents(res *firehose.PutRecordBatchOutput, events []publisher.Event) []publisher.Event {
	if aws.Int64Value(res.FailedPutCount) > 0 {
		failedEvents := make([]publisher.Event, 0)
		responses := res.RequestResponses
		for i, r := range responses {
			if aws.StringValue(r.ErrorCode) != "" {
				failedEvents = append(failedEvents, events[i])
			}
		}
		return failedEvents
	}
	return []publisher.Event{}
}
