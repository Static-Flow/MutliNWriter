package MutliNWriter

import (
	"errors"
	"io"
	"sync"
)

/*
WriteError type returned by Write and ShouldWrite. It includes the standard data returned by a usual write
along with the key for the writer that caused the error.
*/
type WriteError struct {
	n   int
	key any
	err error
}

// WriterKey returns the key for the MultiNWriter child that caused this error
func (w WriteError) WriterKey() any {
	return w.key
}

// BytesWritten returns the number of bytes written by the MultiNWriter child that caused this error
func (w WriteError) BytesWritten() int {
	return w.n
}

// Error returns the underlying io error the MultiNWriter child caused
func (w WriteError) Error() string {
	return w.err.Error()
}

// MultiNWriter is a variant of io.MultiWriter which allows dynamically adding/removing io.Writer's at runtime.
type MultiNWriter struct {
	mutex   sync.Mutex
	writers map[any]io.Writer
}

// NewMultiNWriter create a new instance of MultiNWriter
func NewMultiNWriter() *MultiNWriter {
	return &MultiNWriter{
		mutex:   sync.Mutex{},
		writers: make(map[any]io.Writer),
	}
}

/*
*
Write writes the provided bytes to all children io.Writer's. Any errors during the writes are collected and
returned using errors.Join. The individual `WriteError` errors can be inspected by unwrapping the joined error
like so: ```err.(interface{ Unwrap() []error }).Unwrap()```.
*/
func (mnw *MultiNWriter) Write(input []byte) error {
	var writeErrors []error
	for writerKey, writer := range mnw.writers {
		n, writeErr := writer.Write(input)
		if writeErr != nil {
			writeErrors = append(writeErrors, WriteError{
				n:   n,
				key: writerKey,
				err: writeErr,
			})
		}
	}
	return errors.Join(writeErrors...)
}

/*
ShouldWrite writes the provided bytes to all children io.Writer's. If an error occurs during one of the child writes
ShouldWrite returns early with the error.

Note: Due to the random access of maps, there exists an edge case where if the last writer chosen is the cause of the error
the "return early" is also the end of the function and thus behaves exactly like Write.
*/
func (mnw *MultiNWriter) ShouldWrite(input []byte) error {
	for writerKey, writer := range mnw.writers {
		if bytesWritten, err := writer.Write(input); err != nil {
			return WriteError{
				n:   bytesWritten,
				key: writerKey,
				err: err,
			}
		}
	}
	return nil
}

/*
AddWriter inserts a new io.Writer.
*/
func (mnw *MultiNWriter) AddWriter(key any, writer io.Writer) {
	mnw.mutex.Lock()
	mnw.writers[key] = writer
	mnw.mutex.Unlock()
}

/*
RemoveWriter removed a io.Writer.
*/
func (mnw *MultiNWriter) RemoveWriter(key any) {
	mnw.mutex.Lock()
	delete(mnw.writers, key)
	mnw.mutex.Unlock()
}
