package firehose

import (
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
