// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package filestream

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

func TestLogFileTimedClosing(t *testing.T) {
	testCases := []struct {
		name           string
		createFile     func(t *testing.T) *os.File
		waitBeforeRead time.Duration
		inactive       time.Duration
		closeEOF       bool
		afterInterval  time.Duration
		expectedErr    error
	}{
		{name: "plain: read from file and close inactive",
			createFile:  createTestPlainLogFile,
			inactive:    2 * time.Second,
			expectedErr: ErrInactive,
		},
		{name: "plain: read from file and close after interval",
			createFile:    createTestPlainLogFile,
			afterInterval: 3 * time.Second,
			expectedErr:   ErrClosed,
		},
		{name: "plain: read from file and close on EOF",
			createFile:  createTestPlainLogFile,
			closeEOF:    true,
			expectedErr: io.EOF,
		},
		{name: "GZIP: read from file and close on EOF",
			createFile:  createTestGzipLogFile,
			closeEOF:    true,
			expectedErr: io.EOF,
		},
		{name: "GZIP: read from file and close after interval",
			createFile:     createTestPlainLogFile,
			afterInterval:  3 * time.Second,
			waitBeforeRead: 3 * time.Second,
			expectedErr:    ErrClosed,
		},
	}

	for _, tc := range testCases {
		fs := filestream{
			readerConfig: readerConfig{BufferSize: 512},
			compression:  CompressionAuto}
		f, err := fs.newFile(tc.createFile(t))
		require.NoError(t, err,
			"could not create file for reading")
		defer f.Close()
		defer os.Remove(f.Name())

		t.Run(tc.name, func(t *testing.T) {
			reader, _, err := newFileReader(
				logp.NewNopLogger(),
				context.TODO(),
				f,
				readerConfig{},
				closerConfig{
					OnStateChange: stateChangeCloserConfig{
						CheckInterval: 1 * time.Second,
						Inactive:      tc.inactive,
					},
					Reader: readerCloserConfig{
						OnEOF:         tc.closeEOF,
						AfterInterval: tc.afterInterval,
					},
				},
				false,
			)
			if err != nil {
				t.Fatalf("error while creating logReader: %+v", err)
			}

			if tc.waitBeforeRead > 0 {
				// GZIP files aren't kept open, thus we need to wait for
				// 'AfterInterval' to elapse before reading.
				time.Sleep(tc.waitBeforeRead)
			}

			err = readUntilError(reader)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestLogFileTruncated(t *testing.T) {
	tcs := []struct {
		name       string
		createFile func(t *testing.T) *os.File
		truncateFn func(t *testing.T, f File) error
		wantErr    error
	}{
		{name: "plain: ErrFileTruncate",
			createFile: createTestPlainLogFile,
			truncateFn: func(t *testing.T, f File) error {
				return f.OSFile().Truncate(0)
			},
			wantErr: ErrFileTruncate,
		},
		{name: "gzip: io.EOF",
			createFile: createTestGzipLogFile,
			truncateFn: func(t *testing.T, f File) error {
				osf := f.OSFile()
				gzw := gzip.NewWriter(osf)

				_, err := io.Copy(gzw, bytes.NewBuffer([]byte("truncated data\n")))
				require.NoError(t, err, "could not write data to gzip file")

				require.NoErrorf(t, gzw.Close(),
					"could not close gzip writer")
				require.NoError(t, osf.Sync(), "could not sync os file")

				return nil
			},
			wantErr: io.EOF,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			osFile := tc.createFile(t)

			fs := filestream{
				readerConfig: readerConfig{BufferSize: 512},
				compression:  CompressionAuto}

			f, err := fs.newFile(osFile)
			require.NoError(t, err, "could not create file for reading")

			defer f.Close()
			defer os.Remove(f.Name())

			reader, _, err := newFileReader(
				logp.NewNopLogger(), context.TODO(), f, fs.readerConfig, fs.closerConfig, false)
			require.NoError(t, err, "error while creating logReader")

			buf := make([]byte, 32)
			_, err = reader.Read(buf)
			assert.NoError(t, err)

			err = tc.truncateFn(t, f)
			require.NoError(t, err, "error while truncating file")

			err = readUntilError(reader)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestLogFile_startReadUntilEOF(t *testing.T) {
	// newLogFile builds a logFile with just enough state for
	// startReadUntilEOF to run: a non-nil tg (Stop is called on it). The zero
	// value of unison.TaskGroup is documented as fully functional.
	newLogFile := func() *logFile {
		return &logFile{tg: &unison.TaskGroup{}}
	}

	t.Run("flips closeOnEOF from false to true", func(t *testing.T) {
		lf := newLogFile()
		lf.closeOnEOF = false

		lf.startReadUntilEOF(ctxtool.CancelContext{})

		assert.True(t, lf.closeOnEOF, "startReadUntilEOF must set closeOnEOF to true")
	})

	t.Run("leaves closeOnEOF true if already true", func(t *testing.T) {
		lf := newLogFile()
		lf.closeOnEOF = true

		lf.startReadUntilEOF(ctxtool.CancelContext{})

		assert.True(t, lf.closeOnEOF, "closeOnEOF must remain true")
	})

	t.Run("swaps readerCtx to the passed context", func(t *testing.T) {
		originalCtx := ctxtool.WithCancelContext(context.Background())
		newCtx := ctxtool.WithCancelContext(context.Background())

		lf := newLogFile()
		lf.readerCtx = originalCtx
		lf.startReadUntilEOF(newCtx)

		// Cancelling the original context must NOT cancel readerCtx
		// (it was swapped out).
		originalCtx.Cancel()
		assert.NoError(t, lf.readerCtx.Err(),
			"cancelling the original context must not affect readerCtx after the swap")

		// Cancelling the new context MUST cancel readerCtx.
		newCtx.Cancel()
		assert.Error(t, lf.readerCtx.Err(),
			"cancelling the new (swapped-in) context must cancel readerCtx")
	})

	t.Run("only first call takes effect", func(t *testing.T) {
		firstCtx := ctxtool.WithCancelContext(context.Background())
		secondCtx := ctxtool.WithCancelContext(context.Background())

		lf := newLogFile()
		lf.startReadUntilEOF(firstCtx)
		lf.startReadUntilEOF(secondCtx) // must be a no-op

		// Cancelling secondCtx must not affect readerCtx: the second call
		// was shadowed by sync.Once.
		secondCtx.Cancel()
		require.NoError(t, lf.readerCtx.Err(),
			"secondCtx cancellation leaked into readerCtx; only first startReadUntilEOF call takes effect")

		// Cancelling firstCtx must cancel readerCtx: firstCtx is the one
		// that actually got assigned.
		firstCtx.Cancel()
		require.Error(t, lf.readerCtx.Err(),
			"firstCtx cancellation did not propagate to readerCtx; first call must win")
	})

	// Assert the race-avoidance invariant: by the time
	// startReadUntilEOF returns, any goroutine that was running on
	// f.tg (periodicStateCheck / closeIfTimeout) has exited. Without
	// this, periodicStateCheck could read f.readerCtx and call
	// Cancel() on it concurrently with the swap below.
	t.Run("stops the file-monitoring goroutines before swapping readerCtx",
		func(t *testing.T) {
			lf := newLogFile()

			running := make(chan struct{})
			exited := make(chan struct{})
			err := lf.tg.Go(func(ctx context.Context) error {
				close(running)
				<-ctx.Done()
				close(exited)
				return nil
			})
			require.NoError(t, err, "could not start goroutine on tg")

			// Make sure the goroutine is actually running before we call
			// startReadUntilEOF; otherwise the test could pass even if Stop
			// did nothing.
			<-running

			lf.startReadUntilEOF(ctxtool.WithCancelContext(context.Background()))

			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				t.Fatal("startReadUntilEOF did not stop the tg goroutine")
			}
		})
}

// TestNewFileReader_startReadUntilEOFClosure verifies that the closure
// returned by newFileReader is wired up correctly:
//   - readUntilEOF=true returns a closure bound to (*logFile).startReadUntilEOF,
//     so invoking it swaps the reader's readerCtx and flips closeOnEOF.
//   - readUntilEOF=false returns a no-op closure that leaves the reader
//     untouched, preserving upstream behaviour.
func TestNewFileReader_startReadUntilEOFClosure(t *testing.T) {
	makeReader := func(t *testing.T, readUntilEOF bool) (
		*logFile, func(ctxtool.CancelContext), ctxtool.CancelContext,
	) {
		t.Helper()
		fs := filestream{
			readerConfig: readerConfig{BufferSize: 512},
			compression:  CompressionAuto,
		}
		osFile := createTestPlainLogFile(t)
		t.Cleanup(func() { _ = os.Remove(osFile.Name()) })

		f, err := fs.newFile(osFile)
		require.NoError(t, err, "could not create file for reading")
		t.Cleanup(func() { _ = f.Close() })

		canceler := ctxtool.WithCancelContext(context.Background())
		t.Cleanup(canceler.Cancel)

		reader, enableReadUntilEOFFn, err := newFileReader(
			logp.NewNopLogger(),
			canceler,
			f,
			readerConfig{
				Backoff: backoffConfig{
					Init: 1 * time.Millisecond,
					Max:  10 * time.Millisecond,
				},
			},
			closerConfig{},
			readUntilEOF,
		)
		require.NoError(t, err, "could not create logReader")
		t.Cleanup(func() { _ = reader.Close() })

		return reader, enableReadUntilEOFFn, canceler
	}

	t.Run("readUntilEOF=true: closure swaps readerCtx and set closeOnEOF=true", func(t *testing.T) {
		reader, startReadUntilEOF, _ := makeReader(t, true)

		require.False(t, reader.closeOnEOF,
			"test setup is wrong: closeOnEOF must be false before startReadUntilEOF is called. Did you change the test?")

		newCtx := ctxtool.WithCancelContext(context.Background())
		startReadUntilEOF(newCtx)

		assert.True(t, reader.closeOnEOF,
			"closure must flip closeOnEOF to true")

		// Verify the swap happened: cancelling newCtx must cancel
		// reader.readerCtx.
		newCtx.Cancel()
		assert.Error(t, reader.readerCtx.Err(),
			"closure must swap readerCtx: cancelling newCtx should cancel readerCtx")
	})

	t.Run("readUntilEOF=false: closure is a no-op", func(t *testing.T) {
		reader, startReadUntilEOF, _ := makeReader(t, false)

		require.False(t, reader.closeOnEOF,
			"test setup is wrong: closeOnEOF must be false before startReadUntilEOF is called. Did you change the test?")

		// Capture the readerCtx state before calling the closure.
		originalErr := reader.readerCtx.Err()

		newCtx := ctxtool.WithCancelContext(context.Background())
		startReadUntilEOF(newCtx)

		assert.False(t, reader.closeOnEOF,
			"no-op closure must not set closeOnEOF=true")

		// Cancelling newCtx must not affect reader.readerCtx since the
		// closure is a no-op.
		newCtx.Cancel()
		assert.Equal(t, originalErr, reader.readerCtx.Err(),
			"no-op closure must not swap readerCtx")
	})
}

// TestLogFile_readUntilEOFAfterReaderCtxCancel proves that after readerCtx
// is cancelled by anything that would normally close the reader
// (close.reader.after_interval, close.on_state_change.*, or an explicit
// Close), invoking the readUntilEOF closure lets the reader resume reading
// the file and reach io.EOF. It simulates the cancellation by calling
// reader.readerCtx.Cancel() directly — exactly what closeIfTimeout and
// periodicStateCheck do internally.
func TestLogFile_readUntilEOFAfterReaderCtxCancel(t *testing.T) {
	fs := filestream{
		readerConfig: readerConfig{BufferSize: 512},
		compression:  CompressionAuto,
	}
	osFile := createTestPlainLogFile(t)
	t.Cleanup(func() { _ = os.Remove(osFile.Name()) })

	f, err := fs.newFile(osFile)
	require.NoError(t, err, "could not create file for reading")
	t.Cleanup(func() { _ = f.Close() })

	canceler := ctxtool.WithCancelContext(context.Background())
	t.Cleanup(canceler.Cancel)

	reader, startReadUntilEOF, err := newFileReader(
		logp.NewNopLogger(),
		canceler,
		f,
		readerConfig{
			Backoff: backoffConfig{
				Init: 1 * time.Millisecond,
				Max:  10 * time.Millisecond,
			},
		},
		closerConfig{},
		true, // readUntilEOF enabled
	)
	require.NoError(t, err, "could not create logReader")
	t.Cleanup(func() { _ = reader.Close() })

	// 1. Read the first chunk successfully.
	buf := make([]byte, 16)
	n, err := reader.Read(buf)
	require.NoError(t, err, "first Read must succeed")
	require.Positive(t, n, 0, "first Read must return data")

	// 2. Simulate something else cancelling the reader. closeIfTimeout
	//    (close.reader.after_interval) and periodicStateCheck
	//    (close.on_state_change.*) both do exactly this.
	reader.readerCtx.Cancel()

	// 3. The next Read must return ErrClosed: readerCtx is cancelled and
	//    isInactive was never set.
	_, err = reader.Read(buf)
	require.ErrorIs(t, err, ErrClosed,
		"Read must return ErrClosed after readerCtx.Cancel()")

	// 4. Trigger readUntilEOF mode with a fresh context.
	newCtx := ctxtool.WithCancelContext(context.Background())
	t.Cleanup(newCtx.Cancel)
	startReadUntilEOF(newCtx)

	// 5. Read the rest of the file. The reader must resume successfully and
	//    eventually return io.EOF (closeOnEOF was just set to true).
	finalErr := readUntilError(reader)
	assert.ErrorIs(t, finalErr, io.EOF,
		"after startReadUntilEOF, the reader must finish the file and return io.EOF")
}

// TestNewFileReader_backoffWakesOnCanceler
// regardless of readUntilEOF, input cancellation wakes a parked backoff.
func TestNewFileReader_backoffWakesOnCanceler(t *testing.T) {
	for _, readUntilEOF := range []bool{false, true} {
		name := "readUntilEOF=false"
		if readUntilEOF {
			name = "readUntilEOF=true"
		}

		t.Run(name, func(t *testing.T) {
			fs := filestream{
				readerConfig: readerConfig{BufferSize: 512},
				compression:  CompressionAuto,
			}
			osFile := createTestPlainLogFile(t)
			t.Cleanup(func() { _ = os.Remove(osFile.Name()) })

			f, err := fs.newFile(osFile)
			require.NoError(t, err, "could not create file for reading")
			t.Cleanup(func() { _ = f.Close() })

			canceler := ctxtool.WithCancelContext(context.Background())
			t.Cleanup(canceler.Cancel)

			reader, _, err := newFileReader(
				logp.NewNopLogger(),
				canceler,
				f,
				readerConfig{
					Backoff: backoffConfig{
						Init: 1 * time.Hour,
						Max:  1 * time.Hour,
					},
				},
				closerConfig{},
				readUntilEOF,
			)
			require.NoError(t, err, "could not create logReader")
			t.Cleanup(func() { _ = reader.Close() })

			// Drain the file content (three known lines) so the next Read
			// parks in backoff.Wait.
			readDone := make(chan error, 1)
			go func() {
				readDone <- readUntilError(reader)
			}()

			// Give the reader a moment to drain the file and enter backoff.
			// Cancelling during initial reads is fine too; the invariant we
			// care about is that Read returns promptly after cancel, not
			// after the 1-hour backoff.
			time.Sleep(100 * time.Millisecond)
			canceler.Cancel()

			select {
			case err := <-readDone:
				assert.ErrorIs(t, err, ErrClosed,
					"Read must return ErrClosed once canceler is cancelled")
			case <-time.After(2 * time.Second):
				t.Fatal("Read did not return within 2s after canceler.Cancel(); " +
					"backoff is not wired to canceler.Done()")
			}
		})
	}
}

func createTestPlainLogFile(t *testing.T) *os.File {
	f, err := os.CreateTemp("", "filestream_reader_test")
	require.NoError(t, err, "could not create temp file")

	content := []byte("first log line\nanother interesting line\na third log message\n")
	_, err = f.Write(content)
	require.NoError(t, err, "could not write to temp file")

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err, "could not seek to start of temp file")

	return f
}

func createTestGzipLogFile(t *testing.T) *os.File {
	plain := createTestPlainLogFile(t)

	f, err := os.CreateTemp("", "filestream_reader_test.*.gz")
	require.NoError(t, err, "could not create temp file")

	data, err := io.ReadAll(plain)
	require.NoError(t, err, "could not read from file")

	var tempBuffer bytes.Buffer
	gw := gzip.NewWriter(&tempBuffer)
	_, err = gw.Write(data)
	require.NoError(t, err, "failed to write plain content to gzip writer")
	err = gw.Close()
	require.NoError(t, err, "failed to close gzip writer")

	_, err = f.Write(tempBuffer.Bytes())
	require.NoError(t, err, "failed to write to gzip file")

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err, "could not seek to start of gzip file")

	return f
}

func readUntilError(reader *logFile) error {
	buf := make([]byte, 1024)
	_, err := reader.Read(buf)
	for err == nil {
		buf := make([]byte, 1024)
		_, err = reader.Read(buf)
	}
	return err
}
