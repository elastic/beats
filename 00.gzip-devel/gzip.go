package gzip_devel

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

var lineText func() string
var sizes = []int{
	1 * humanize.MiByte,
	5 * humanize.MiByte,
	25 * humanize.MiByte,
	50 * humanize.MiByte,
	100 * humanize.MiByte,
	250 * humanize.MiByte,
	500 * humanize.MiByte,
	1000 * humanize.MiByte,
}

func BenchmarkGzip(b *testing.B) {
	publisher := noopPublisher{}

	tcs := []struct {
		name   string
		lineFn func() string
	}{
		{name: "static-line", lineFn: func() string { return "line" }},
		{name: "random-line", lineFn: rand.Text},
	}
	for _, tc := range tcs {
		lineText = tc.lineFn

		b.Run(tc.name, func(b *testing.B) {
			for _, size := range sizes {
				plainFile, gzFile, lines := createLogFile(
					b, calculateLines(size))

				half := lines / 2
				leftOver := half + (lines % 2)

				// b.Run(fmt.Sprintf("%s-plain-file", humanize.Bytes(uint64(size))),
				// 	func(b *testing.B) {
				// 		for i := 0; i < b.N; i++ {
				// 			readPlainFile(b, plainFile, half, leftOver, publisher)
				// 		}
				// 	})
				//
				// b.Run(fmt.Sprintf("%s-gzip-file", humanize.Bytes(uint64(size))),
				// 	func(b *testing.B) {
				// 		for i := 0; i < b.N; i++ {
				// 			readGZFile(b, gzFile, half, leftOver, publisher)
				// 		}
				// 	})

				b.Run(fmt.Sprintf("%s", humanize.Bytes(uint64(size))),
					func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							// readPlainFile(b, plainFile, half, leftOver, publisher)
							readGZFile(b, gzFile, half, leftOver, publisher)
						}
					})

				err := os.Remove(plainFile)
				assert.NoError(b, err, "could not delete %s file %s", plainFile)
				err = os.Remove(gzFile)
				assert.NoError(b, err, "could not delete %s file %s", gzFile)
			}
		})
	}
}

func TestTimeDifferenceGzipPlain(t *testing.T) {
	publisher := noopPublisher{}

	tcs := []struct {
		name   string
		lineFn func() string
	}{
		{name: "static-line", lineFn: func() string { return "line" }},
		{name: "random-line", lineFn: rand.Text},
	}
	for _, tc := range tcs {
		lineText = tc.lineFn
		t.Run(tc.name, func(t *testing.T) {
			for _, size := range sizes {

				plainFile, gzFile, lines := createLogFile(
					t, calculateLines(size))

				half := lines / 2
				leftOver := half + (lines % 2)

				var plainDuration, gzipDuration time.Duration
				t.Run(fmt.Sprintf("%s-plain-file", humanize.Bytes(uint64(size))),
					func(b *testing.T) {
						now := time.Now()
						readPlainFile(b, plainFile, half, leftOver, publisher)
						plainDuration = time.Since(now)
						b.Logf("readPlainFile took %s for %d lines", plainDuration, lines)
					})

				t.Run(fmt.Sprintf("%s-gzip-file", humanize.Bytes(uint64(size))),
					func(b *testing.T) {
						now := time.Now()
						readGZFile(b, gzFile, half, leftOver, publisher)
						gzipDuration = time.Since(now)
						b.Logf("readGZFile took %s for %d lines", gzipDuration, lines)
					})

				t.Logf("\n%s: for %s, gzip read took %s more than pain read\n\n",
					tc.name,
					humanize.Bytes(uint64(size)),
					gzipDuration-plainDuration)

				err := os.Remove(plainFile)
				assert.NoError(t, err, "could not delete %s file %s", plainFile)
				err = os.Remove(gzFile)
				assert.NoError(t, err, "could not delete %s file %s", gzFile)
			}
		})
	}
}

func TestGzip(t *testing.T) {
	_, gzFile, lines := createLogFile(t, 42)

	publisher := &debugPublisher{}
	half := lines / 2
	leftOver := half + (lines % 2)

	readGZFile(t, gzFile, half, leftOver, publisher)
}

// ================================ Publishers ================================
type Publisher interface {
	Publish([]byte)
}

type noopPublisher struct{}

func (n noopPublisher) Publish([]byte) {}

type debugPublisher struct {
	counter int
}

func (p *debugPublisher) Publish(data []byte) {
	fmt.Println(string(data))
	p.counter++
}

// ============================= gzipSeekerReader =============================

type gzipSeekerReader struct {
	f   *os.File
	gzr *gzip.Reader
}

func (r *gzipSeekerReader) Read(p []byte) (n int, err error) {
	return r.gzr.Read(p)
}

func (r *gzipSeekerReader) Close() error {
	return r.gzr.Close()
}

func (r *gzipSeekerReader) Seek(offset int64, whence int) (int64, error) {
	if whence != 0 {
		return 0, fmt.Errorf("only 0 is allowed for whence, got: %d", whence)
	}

	// Not used in the test/benchmark
	if offset == 0 {
		n, err := r.f.Seek(0, 0)
		if err != nil {
			return n, fmt.Errorf("gzipSeekerReader: could not seek to 0: %w", err)
		}

		if err = r.gzr.Close(); err != nil {
			return n, fmt.Errorf(
				"gzipSeekerReader: could not close gzip reader before creating a new one: %w", err)
		}
		r.gzr, err = gzip.NewReader(r.f)
		if err != nil {
			return n, fmt.Errorf("gzipSeekerReader: could not create new gzip reader: %w", err)
		}
		return n, nil
	}

	var n int
	var err error
	buffSize := int64(512)
	if offset <= buffSize {
		n, err = r.Read(make([]byte, offset))
		if err != nil {
			return int64(n), fmt.Errorf("gzipSeekerReader: could read offset=%d: %w", offset, err)
		}
		return int64(n), nil
	}

	chunks := offset / buffSize
	leftover := offset % buffSize
	buff := make([]byte, buffSize)
	read := 0
	for i := range chunks {
		n, err = r.gzr.Read(buff)
		read += n
		if err != nil {
			return int64(read), fmt.Errorf("gzipSeekerReader: could read chunk %d: %w", i, err)
		}
	}

	if leftover > 0 {
		n, err = r.Read(make([]byte, leftover))
		read += n
		if err != nil && err != io.EOF {
			return int64(read), fmt.Errorf("gzipSeekerReader: could read leftover %d: %w", leftover, err)
		}
	}

	return offset, nil
}

// ============================= helper functions =============================

func readPlainFile(
	t testing.TB, path string, half, rest int, publisher Publisher) {

	f1, err := os.Open(path)
	require.NoErrorf(t, err, "could not open log file %s", path)
	defer f1.Close()

	f2, err := os.Open(path)
	require.NoErrorf(t, err, "could not open log file %s", path)
	defer f2.Close()

	readInHalfThenHalf(t, half, rest, f1, f2, publisher)
}

func readGZFile(
	t testing.TB, path string, half, rest int, publisher Publisher) {

	gzSeekReader1, f1, gzr1 := newGZSeekReader(t, path)
	defer gzr1.Close()
	defer f1.Close()

	gzSeekReader2, f2, gzr2 := newGZSeekReader(t, path)
	defer gzr2.Close()
	defer f2.Close()

	readInHalfThenHalf(t, half, rest, gzSeekReader1, gzSeekReader2, publisher)
}

func newGZSeekReader(t testing.TB, path string) (*gzipSeekerReader, *os.File, *gzip.Reader) {
	f, err := os.Open(path)
	require.NoErrorf(t, err, "could not open log file %s", path)

	gzr, err := gzip.NewReader(f)
	require.NoErrorf(t, err, "could not create gzip reader")
	gzSeekReader := &gzipSeekerReader{f: f, gzr: gzr}

	return gzSeekReader, f, gzr
}

func readInHalfThenHalf(
	t testing.TB,
	half int,
	rest int,
	reader1 io.ReadSeekCloser,
	reader2 io.ReadSeekCloser,
	publisher Publisher) {

	offset, err := readNLines(t, reader1, half, publisher)

	_, err = reader2.Seek(int64(offset), 0)
	require.NoErrorf(t, err, "could not seek")

	offset, err = readNLines(t, reader2, rest, publisher)
	require.NoError(t, err)
}

func readNLines(t testing.TB, f io.ReadCloser, n int, publisher Publisher) (int, error) {
	codec, err := encoding.Plain(f)
	require.NoError(t, err, "could not create encoder")

	var r reader.Reader
	r, err = readfile.NewEncodeReader(f, readfile.Config{
		Codec:      codec,
		BufferSize: 4096,
		Terminator: readfile.AutoLineTerminator,
	})
	require.NoError(t, err)
	r = readfile.NewStripNewline(r, readfile.AutoLineTerminator)

	var m reader.Message
	offset := 0
	linesRead := 0
	for range n {
		m, err = r.Next()
		if err != nil {
			break
		}
		offset += m.Offset + m.Bytes
		linesRead++
		publisher.Publish(m.Content)
	}

	if linesRead != n {
		err = errors.Join(err, fmt.Errorf("expected %d lines read, got %d",
			n, offset))
	}
	return offset, err
}

// calculateLines calculates the approximate number of lines to reach the
// size in bytes. Tip, use humanize.KiByte, humanize.MiByte, and so on.
func calculateLines(sizeInBytes int) int {
	text := []byte(lineText() + " xxxxxxxxx")
	return int(math.Ceil(float64(sizeInBytes) / float64(len(text))))
}

func createLogFile(t testing.TB, lines int) (string, string, int) {
	filename := filepath.Join(t.TempDir(), "log.log")
	gzFilename := filename + ".gz"

	f, err := os.Create(filename)
	require.NoErrorf(t, err, "could not create log file")

	for i := range lines {
		// _, err = fmt.Fprintln(f, "line "+strconv.Itoa(i))
		_, err = fmt.Fprintf(f, "line %d - %s\n", i, lineText())
		require.NoErrorf(t, err, "could not write line %d/%d to log file",
			i, lines)
	}
	require.NoErrorf(t, f.Close(), "could not close log file")

	plainFile, err := os.Open(filename)
	stat, err := plainFile.Stat()
	require.NoErrorf(t, err, "could not stat log file")
	t.Logf("plain file to be gzipped has %s", humanize.Bytes(uint64(stat.Size())))

	gzipFile, err := os.Create(gzFilename)
	require.NoErrorf(t, err, "could not open log file")

	gzw := gzip.NewWriter(gzipFile)

	_, err = io.Copy(gzw, plainFile)
	require.NoErrorf(t, err, "could write date to gzip file")
	require.NoErrorf(t, gzw.Close(), "could not close gzip writer")
	require.NoErrorf(t, gzipFile.Close(), "could not close gzip file")
	assert.NoErrorf(t, plainFile.Close(), "could not close plain file used of for reading")

	return filename, gzFilename, lines
}

func TestCorruptedGZIPFile(t *testing.T) {
	lines := 10
	plainBuff := &bytes.Buffer{}
	for i := range lines {
		_, err := fmt.Fprintf(plainBuff, "line %d - %s\n", i, rand.Text())
		require.NoErrorf(t, err, "could not write line %d/%d to log buffer",
			i, lines)
	}

	folder := filepath.Join("/", "tmp", "beats")

	// lineText = rand.Text
	// _, gzFile, lines := createLogFile(
	// 	t, calculateLines(1000*humanize.MiByte))
	// fmt.Printf("file with %d lines created\n", lines)
	//
	//
	// // move the gzFile to the tmp folder
	// err := os.Rename(gzFile, filepath.Join(folder, filepath.Base(gzFile)))
	// require.NoError(t, err, "could not move gzFile to tmp folder")
	// fmt.Println("gzFile moved to tmp folder")

	valid, corrupted := craftCorruptedGzip(t, plainBuff.Bytes())
	err := os.WriteFile(filepath.Join(folder, "crc-size-check-valid.gz"), valid, 0644)
	require.NoError(t, err, "could not write valid gzip file")

	err = os.WriteFile(filepath.Join(folder, "crc-size-check-corrupted.gz"), corrupted, 0644)
	require.NoError(t, err, "could not write corrupted gzip file")
}

// craftCorruptedGzip takes input data, compresses it using gzip,
// and then intentionally corrupts the footer (CRC32 and ISIZE)
// to simulate checksum/length errors upon decompression.
// It returns the valid, compressed, GZIP and the corrupted version.
// Check the RFC1 952 for details https://www.rfc-editor.org/rfc/rfc1952.html.
func craftCorruptedGzip(t *testing.T, data []byte) ([]byte, []byte) {
	fmt.Println("len(data):", len(data))

	var gzBuff bytes.Buffer
	gw := gzip.NewWriter(&gzBuff)

	wrote, err := gw.Write(data)
	require.NoError(t, err, "failed to write data to gzip writer")
	// sanity check
	require.Equal(t, len(data), wrote, "written data is not equal to input data")
	require.NoError(t, gw.Close(), "failed to close gzip writer")

	compressedBytes := gzBuff.Bytes()
	var validGZ = make([]byte, len(compressedBytes))
	copied := copy(validGZ, compressedBytes)
	require.Equal(t, len(compressedBytes), copied, "copied bytes is not equal to input bytes")

	// get the footer start index
	footerStartIndex := len(compressedBytes) - 8

	// CRC32 - first 4 bytes of footer
	originalCRC32 := binary.LittleEndian.Uint32(compressedBytes[footerStartIndex : footerStartIndex+4])
	fmt.Println("Original CRC32:", originalCRC32)

	// corrupted the CRC32, anything will do.
	corruptedCRC32 := originalCRC32 + 1
	binary.LittleEndian.PutUint32(compressedBytes[footerStartIndex:footerStartIndex+4], corruptedCRC32)
	fmt.Println("Corrupted CRC32:", corruptedCRC32)

	// ISIZE - last 4 bytes of footer
	originalISIZE := binary.LittleEndian.Uint32(compressedBytes[footerStartIndex+4 : footerStartIndex+8])
	fmt.Println("Original ISIZE:", originalISIZE)
	// corrupted the ISIZE, anything will do
	corruptedISIZE := originalISIZE + 1
	binary.LittleEndian.PutUint32(compressedBytes[footerStartIndex+4:footerStartIndex+8], corruptedISIZE)
	fmt.Println("Corrupted ISIZE:", corruptedISIZE)

	assert.Equal(t, validGZ, compressedBytes, "compressed data is not equal to compressed data")
	return validGZ, compressedBytes
}
