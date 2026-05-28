package logger

import (
	"net/http"
	"sync/atomic"
	"testing"
)

type countingRoundTripper struct {
	received atomic.Int64
}

func (rt *countingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.received.Add(1)
	return &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

func TestSeqAsyncWriterCloseFlushesBufferedLogs(t *testing.T) {
	transport := &countingRoundTripper{}
	writer := NewSeqAsyncWriter("http://seq.test")
	writer.client = &http.Client{Transport: transport}

	for range 10 {
		if _, err := writer.Write([]byte(`{"@m":"test"}`)); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}

	if got := transport.received.Load(); got != 10 {
		t.Fatalf("received %d logs, want 10", got)
	}

	if n, err := writer.Write([]byte(`{"@m":"after close"}`)); err != nil || n == 0 {
		t.Fatalf("Write after Close returned n=%d err=%v", n, err)
	}
	if got := transport.received.Load(); got != 10 {
		t.Fatalf("received %d logs after write on closed writer, want 10", got)
	}
}
