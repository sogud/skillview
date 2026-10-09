package updates

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha512"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func downloadRelease(url string, size int) *updater.Release {
	return &updater.Release{
		Artifact:     updater.Artifact{Size: int64(size)},
		Metadata:     map[string]any{"endpoint.artifact.url": url, "skillview.manifestURL": url + "/manifest.json"},
		Verification: &updater.Verification{DigestAlgo: "sha512", Digest: make([]byte, sha512.Size), SignatureAlgo: "ed25519ph", Signature: make([]byte, ed25519.SignatureSize)},
	}
}

func TestDownloadResumesAfterConnectionClosesHalfway(t *testing.T) {
	payload := []byte(strings.Repeat("signed archive bytes", 100))
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
			_, _ = w.Write(payload[:len(payload)/2])
			w.(http.Flusher).Flush()
			conn, _, _ := w.(http.Hijacker).Hijack()
			_ = conn.Close()
			return
		}
		if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", len(payload)/2) {
			t.Errorf("resume offset: %q", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", len(payload)/2, len(payload)-1, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[len(payload)/2:])
	}))
	defer server.Close()
	provider, err := newProvider(server.Client(), server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := provider.Download(context.Background(), downloadRelease(server.URL, len(payload)), &output, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), payload) || requests != 2 {
		t.Fatalf("received %d bytes in %d requests", output.Len(), requests)
	}
}

func TestArtifactDownloadOutlivesShortCheckTimeout(t *testing.T) {
	payload := []byte("complete signed archive")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write(payload[:5])
		w.(http.Flusher).Flush()
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write(payload[5:])
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 25 * time.Millisecond
	provider, err := newProvider(client, server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := provider.Download(context.Background(), downloadRelease(server.URL, len(payload)), &output, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), payload) {
		t.Fatal("download was truncated")
	}
}

func TestResumeRejectsWrongRangeWithoutAppendingDuplicateBytes(t *testing.T) {
	for _, mode := range []string{"ignored range", "wrong offset"} {
		t.Run(mode, func(t *testing.T) {
			payload := []byte("complete archive")
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests == 1 {
					w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
					_, _ = w.Write(payload[:5])
					w.(http.Flusher).Flush()
					conn, _, _ := w.(http.Hijacker).Hijack()
					_ = conn.Close()
					return
				}
				if mode == "wrong offset" {
					w.Header().Set("Content-Range", "bytes 0-15/16")
					w.WriteHeader(206)
				}
				_, _ = w.Write(payload)
			}))
			defer server.Close()
			provider, _ := newProvider(server.Client(), server.URL, true)
			var output bytes.Buffer
			if err := provider.Download(context.Background(), downloadRelease(server.URL, len(payload)), &output, nil); err == nil || !bytes.Equal(output.Bytes(), payload[:5]) {
				t.Fatal("unsafe range appended", err, output.Len())
			}
		})
	}
}

func TestDownloadCancellationStopsRetries(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Length", "20")
		_, _ = w.Write([]byte("first"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	provider, _ := newProvider(server.Client(), server.URL, true)
	ctx, cancel := context.WithCancel(context.Background())
	var output bytes.Buffer
	err := provider.Download(ctx, downloadRelease(server.URL, 20), &output, func(int64, int64) { cancel() })
	if !errors.Is(err, context.Canceled) || requests != 1 {
		t.Fatal("cancellation retried", err, requests)
	}
}
