package main

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/unixpickle/essentials"
)

type Header struct {
	ClientTag string
	Url       string
}

type Message struct {
	CommuniqueType string
	Header         Header
	Body           json.RawMessage `json:",omitempty"`
}

type BrokerConn interface {
	Send(_ Message) error
	Subscribe(_ context.Context, _ chan<- Message, _ ...func() error) error
	Call(_ context.Context, _ Message, _ func(Message) (bool, error)) (Message, error)
	Close() error
	Error() error
}

// ReadRequest sends a ReadRequest to the given URL and parses the result into
// a specified JSON object `result`.
func ReadRequest(ctx context.Context, conn BrokerConn, url string, result any) (err error) {
	defer essentials.AddCtxTo("request "+url, &err)

	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	clientTag := uuid.String()
	msg := Message{
		CommuniqueType: "ReadRequest",
		Header: Header{
			ClientTag: clientTag,
			Url:       url,
		},
	}
	response, err := conn.Call(ctx, msg, func(response Message) (bool, error) {
		return response.Header.ClientTag == clientTag, nil
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(response.Body), result)
}
