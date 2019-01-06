package firehose

import (
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/publisher"
	"testing"
)

type MockCodec struct {
}

func (mock MockCodec) Encode(index string, event *beat.Event) ([]byte, error) {
	return []byte("boom"), nil
}

func TestMapEvent(t *testing.T) {
	client := client{encoder: MockCodec{}}
	record, _ := client.mapEvent(&publisher.Event{})

	if string(record.Data) != "boom\n" {
		t.Errorf("Unexpected data: %s", record.Data)
	}
}

func TestMapEvents(t *testing.T) {
	client := client{encoder: MockCodec{}}
	events := []publisher.Event{{}}
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

func TestCollectFailedEvents(t *testing.T) {
	client := client{encoder: MockCodec{}}
	events := []publisher.Event{{}, {}}
	okEvents, _, _ := client.mapEvents(events)

	res := firehose.PutRecordBatchOutput{}
	entry1 := firehose.PutRecordBatchResponseEntry{}
	entry2 := firehose.PutRecordBatchResponseEntry{}
	responses := []*firehose.PutRecordBatchResponseEntry{&entry1, &entry2}
	res.SetRequestResponses(responses)

	{
		failed := collectFailedEvents(&res, okEvents)

		if len(failed) != 0 {
			t.Errorf("Expected 0 failed, got %v", len(failed))
		}
	}
	{
		res.SetFailedPutCount(1)
		entry2.SetErrorCode("boom")

		failed := collectFailedEvents(&res, okEvents)

		if len(failed) != 1 {
			t.Errorf("Expected 1 failed, got %v", len(failed))
		}
	}
}

func TestClient_String(t *testing.T) {
	client := client{encoder: MockCodec{}}

	if v := client.String(); v != "firehose" {
		t.Errorf("unexpected value '%v'", v)
	}
}
