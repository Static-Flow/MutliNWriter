package MultiNWriter

import (
	"errors"
	"io"
	"sync"
)

type writeError struct {
	n   int
	key any
	err error
}

func (w writeError) WriterKey() any {
	return w.key
}

func (w writeError) BytesWritten() int {
	return w.n
}

func (w writeError) Error() string {
	return w.err.Error()
}

type MultiNWriter struct {
	mutex   sync.Mutex
	writers map[any]io.Writer
}

func NewMultiNWriter() *MultiNWriter {
	return &MultiNWriter{
		mutex:   sync.Mutex{},
		writers: make(map[any]io.Writer),
	}
}

func (mnw *MultiNWriter) Write(input []byte) error {
	var writeErrors []error
	for writerKey, writer := range mnw.writers {
		n, writeErr := writer.Write(input)
		if writeErr != nil {
			writeErrors = append(writeErrors, writeError{
				n:   n,
				key: writerKey,
				err: writeErr,
			})
		}
	}
	return errors.Join(writeErrors...)
}

func (mnw *MultiNWriter) ShouldWrite(input []byte) error {
	for writerKey, writer := range mnw.writers {
		if bytesWritten, err := writer.Write(input); err != nil {
			return writeError{
				n:   bytesWritten,
				key: writerKey,
				err: err,
			}
		}
	}
	return nil
}

func (mnw *MultiNWriter) AddWriter(key any, writer io.Writer) {
	mnw.mutex.Lock()
	mnw.writers[key] = writer
	mnw.mutex.Unlock()
}

func (mnw *MultiNWriter) RemoveWriter(key any) {
	mnw.mutex.Lock()
	delete(mnw.writers, key)
	mnw.mutex.Unlock()
}
