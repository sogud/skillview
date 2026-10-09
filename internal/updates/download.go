package updates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

const interruptedMessage = "The download was interrupted. Retry the download; your current app is unchanged."
const resumeMessage = "The server could not resume this download. Retry the download from the beginning."

// Append only the exact signed byte range. Wails verifies the complete archive
// digest and signature before extraction, including bytes received after retry.
func (p *releaseProvider) download(ctx context.Context, release *updater.Release, dst io.Writer, progress func(int64, int64)) error {
	address, _ := release.Metadata["endpoint.artifact.url"].(string)
	total := release.Artifact.Size
	if address == "" || total <= 0 {
		return errors.New(invalidReleaseMessage)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	client := *p.client
	client.Timeout = 0 // A check timeout must not cut off a progressing download.
	written := int64(0)
	buffer := make([]byte, 64*1024)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			if p.log != nil {
				p.log("download_retry", fmt.Sprintf("attempt=%d bytes=%d total=%d", attempt+1, written, total))
			}
			timer := time.NewTimer(time.Duration(attempt) * 250 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return errors.New(invalidReleaseMessage)
		}
		request.Header.Set("Accept", "application/octet-stream")
		if written > 0 {
			request.Header.Set("Range", fmt.Sprintf("bytes=%d-", written))
		}
		response, err := client.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var status *responseError
			if errors.As(err, &status) && status.status < 500 && status.status != 429 {
				return err
			}
			if p.log != nil {
				p.log("download_connection", diagnosticReason(err))
			}
			continue
		}
		valid := response.StatusCode == http.StatusOK && written == 0
		if response.StatusCode == http.StatusPartialContent {
			expected := fmt.Sprintf("bytes %d-%d/%d", written, total-1, total)
			valid = response.Header.Get("Content-Range") == expected
		}
		if !valid || (response.ContentLength >= 0 && response.ContentLength != total-written) {
			response.Body.Close()
			return errors.New(resumeMessage)
		}
		for written < total {
			chunk := buffer
			if remaining := total - written; remaining < int64(len(chunk)) {
				chunk = chunk[:remaining]
			}
			n, readErr := response.Body.Read(chunk)
			if n > 0 {
				count, writeErr := dst.Write(chunk[:n])
				if writeErr != nil {
					response.Body.Close()
					return writeErr
				}
				if count != n {
					response.Body.Close()
					return io.ErrShortWrite
				}
				written += int64(n)
				if progress != nil {
					progress(written, total)
				}
			}
			if written == total {
				response.Body.Close()
				return nil
			}
			if readErr != nil {
				if p.log != nil {
					p.log("download_interrupted", diagnosticReason(readErr))
				}
				break
			}
		}
		response.Body.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return errors.New(interruptedMessage)
}
