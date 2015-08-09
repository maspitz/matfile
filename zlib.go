// Copyright 2015 Michael Spitznagel.
// This is program is free software.  You may distribute it under the
// terms of the GNU General Public License.

package matfile

import (
	"compress/zlib"
	"errors"
	"io"
	"sync"
)


// number of bytes to initially read 
const zrBufferInit = 256

// zlibReaderAt applies an uncompressing filter to an io.Reader and
// wraps it with an implementation of io.ReaderAt.
type zlibReaderAt struct {
	r io.ReadCloser
	buf []byte
	lock sync.Mutex
	sourceLength int
	copiedAll bool
	closed bool
}

// TODO find a way to ensure that Close gets called
// (or justify it not having to be closed...)

// Returns a new ReaderAt which provides uncompressed data.
// rd should be a Reader of the zlib-compressed data stream.
// sourceLength should be the total length in bytes of the uncompressed data.
// It is the responsibility of the caller to close the returned ReaderAt.
func newzlibReaderAt(rd io.Reader, sourceLength int) (io.ReaderAt, error) {
	var zat zlibReaderAt
	zat.lock.Lock()
	defer zat.lock.Unlock()

	zat.sourceLength = sourceLength
	if sourceLength <= zrBufferInit {
		zat.buf = make([]byte, sourceLength)
	} else {
		zat.buf = make([]byte, zrBufferInit)
	}

	var err error
	zat.r, err = zlib.NewReader(rd)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(zat.r, zat.buf)
	if err != nil {
		return nil, err
	}
	if sourceLength <= zrBufferInit {
		zat.copiedAll = true
	}
	return zat, nil
}

func (z zlibReaderAt) ReadAt(p []byte, off int64) (int, error) {
	var err error
	z.lock.Lock()
	defer z.lock.Unlock()
	if z.closed == true {
		return 0, io.ErrClosedPipe
	}
	if off < 0 {
		return 0, errors.New("zlibReaderAt.ReadAt: negative offset")
	}
	if off >= int64(z.sourceLength) {
		return 0, io.EOF
	}
	if z.copiedAll == false && off + int64(len(p)) > int64(len(z.buf)) {
		fullBuf := make([]byte, z.sourceLength)
		copy(fullBuf, z.buf)
		_, err = io.ReadFull(z.r, fullBuf[zrBufferInit:])
		if err != nil {
			return 0, err
		}
		z.copiedAll = true
	}
	n := copy(p, z.buf[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (z *zlibReaderAt) Close() error {
	z.lock.Lock()
	z.closed = true
	defer z.lock.Unlock()
	return z.r.Close()
}
