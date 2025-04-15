package main

import (
	"context"
	"encoding/json"
	"sync"

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

// ReadRequests is a concurrent version of ReadRequest() to do batch reads.
func ReadRequests[T any](ctx context.Context, conn BrokerConn, urls []string, results []T) (err error) {
	if len(urls) == 0 {
		return nil
	}
	defer essentials.AddCtxTo("request multiple URLs", &err)

	if len(urls) != len(results) {
		panic("number of URLs must match number of results")
	}

	errChan := make(chan error, len(urls))
	wg := sync.WaitGroup{}
	for i := range urls {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := ReadRequest(ctx, conn, urls[i], &results[i]); err != nil {
				errChan <- err
			}
		}(i)
	}
	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// ReadRequestsAsMap is like ReadRequests but takes in a set and returns a
// mapping.
func ReadRequestsAsMap[T any](ctx context.Context, conn BrokerConn, urls map[string]struct{}) (results map[string]T, err error) {
	orderedURLs := make([]string, 0, len(urls))
	for url := range urls {
		orderedURLs = append(orderedURLs, url)
	}
	orderedOut := make([]T, len(orderedURLs))
	if err := ReadRequests(ctx, conn, orderedURLs, orderedOut); err != nil {
		return nil, err
	}
	results = map[string]T{}
	for i, url := range orderedURLs {
		results[url] = orderedOut[i]
	}
	return results, nil
}
