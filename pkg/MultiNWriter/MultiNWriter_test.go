package MultiNWriter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
)

/*
Simple test to ensure child writers can be added
*/
func TestInsertKeyMultiNWriter(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	multiNWriter.AddWriter("foo", &strings.Builder{})
	if _, ok := multiNWriter.writers["foo"]; !ok {
		t.Errorf("Key foo not found")
	}
}

/*
Simple test to ensure child writers can be removed
*/
func TestRemoveKeyMultiNWriter(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	multiNWriter.AddWriter("foo", &strings.Builder{})
	multiNWriter.RemoveWriter("foo")
	if _, ok := multiNWriter.writers["foo"]; ok {
		t.Errorf("Key foo still exists")
	}
}

/*
Simple test that a child writer receives the parent write
*/
func TestSuccessfullWrite(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	builder := &strings.Builder{}
	multiNWriter.AddWriter("foo", builder)
	err := multiNWriter.Write([]byte("test"))
	if err != nil {
		t.Errorf("Failed to write to foo: %s\n", err.Error())
	}
	if builder.String() != "test" {
		t.Errorf("Failed to write string")
	}
}

/*
This test checks that writes with multiple child writers still write to all healthy ones even if a child writer fails. Two child writers are made,
then a first write, followed by closing the first child writer and trying again. The second writer should receive both writes.
*/
func TestFailedWrite(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	reader, writer := io.Pipe()
	reader2, writer2 := io.Pipe()
	buf := make([]byte, 5)
	buf2 := make([]byte, 5)
	go func() {
		bufferedReader := bufio.NewReader(reader)
		for {
			n, _ := bufferedReader.Read(buf)
			if n > 0 {
				fmt.Println("reader ", string(buf[:n]))
			}
		}
	}()

	go func() {
		bufferedReader := bufio.NewReader(reader2)
		for {
			n, _ := bufferedReader.Read(buf2)
			if n > 0 {
				fmt.Println("reader2 ", string(buf2[:n]))
			}
		}
	}()

	multiNWriter.AddWriter("foo", writer)
	multiNWriter.AddWriter("bar", writer2)

	err := multiNWriter.Write([]byte("test1"))
	if err != nil {
		t.Errorf("Failed to write: %s\n", fmt.Errorf("%w", err))
	} else {
		t.Log("Wrote successfully")
	}
	t.Log("Closing writer foo")
	_ = writer.Close()
	t.Log("trying write again")
	err = multiNWriter.Write([]byte("test2"))
	if err == nil {
		t.Errorf("Write should not succeed on closed writer")
	}

	writeErrors := err.(interface{ Unwrap() []error }).Unwrap()
	if len(writeErrors) != 1 {
		t.Errorf("Only writer foo should have an error")
	} else {
		t.Logf("Only 1 writer failed with error %s with %d bytes writen", writeErrors[0].(writeError).Error(), writeErrors[0].(writeError).BytesWritten())
	}
	if string(buf2) != "test2" {
		t.Errorf("MultiNWriter did not successfully write to bar twice, wanted `test2`, got %s", buf2)
	} else {
		t.Log("Writer bar received both writes")
	}

}

/*
This test checks that calling `ShouldWrite` exits early when a Writer fails. It sets up 10 writers, closes the second one early, then sends their writes to
a channel. When exiting early, there should be less than 9 writes.

NOTE: There is an edge case where if the failed writer is the last to be chosen for writes then it doesn't exit early and behaves just list Write.
*/
func TestFailedShouldWrite(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	resultsChannel := make(chan string, 10)
	waitGroup := sync.WaitGroup{}
	doneWriting := false
	readWriteFunc := func(index int) *io.PipeWriter {
		reader, writer := io.Pipe()
		buf := make([]byte, 5)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			bufferedReader := bufio.NewReader(reader)
			for {
				if doneWriting {
					break
				}
				n, err := bufferedReader.Read(buf)
				if err != nil {
					break
				}
				if n > 0 {
					resultsChannel <- fmt.Sprintf("reader %d %s\n", index, string(buf[:n]))

				}
			}
		}(&waitGroup)
		return writer
	}

	for i := 0; i < 10; i++ {
		waitGroup.Add(1)
		writer := readWriteFunc(i)
		if i == 2 {
			_ = writer.Close()
		}
		go func(w *io.PipeWriter) {
			for {
				if doneWriting {
					_ = w.Close()
				}
			}
		}(writer)
		multiNWriter.AddWriter(i, writer)
	}

	err := multiNWriter.ShouldWrite([]byte("test1"))
	doneWriting = true
	if err == nil {
		t.Errorf("Writer 2 should have failed")
	} else if err.(writeError).WriterKey() != 2 {
		t.Errorf("Only writer 2 should have failed")
	}
	waitGroup.Wait()
	close(resultsChannel)
	resultCount := 0
	for result := range resultsChannel {
		resultCount++
		fmt.Println(result)
	}
	if resultCount == 9 {
		t.Error("Expected less than 9 reads")
	}
	t.Logf("Only got %d reads", resultCount)
}

/*
This test checks that calling `ShouldWrite` exits early when a Writer fails. It sets up 10 writers, closes the second one early, then sends their writes to
a channel. When exiting early, there should be less than 9 writes.

NOTE: There is an edge case where if the failed writer is the last to be chosen for writes then it doesn't exit early and behaves just list Write.
*/
func TestSuccessfulShouldWrite(t *testing.T) {
	multiNWriter := NewMultiNWriter()
	resultsChannel := make(chan string, 10)
	waitGroup := sync.WaitGroup{}
	doneWriting := false
	readWriteFunc := func(index int) *io.PipeWriter {
		reader, writer := io.Pipe()
		buf := make([]byte, 5)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			bufferedReader := bufio.NewReader(reader)
			for {
				if doneWriting {
					break
				}
				n, err := bufferedReader.Read(buf)
				if err != nil {
					break
				}
				if n > 0 {
					resultsChannel <- fmt.Sprintf("reader %d %s\n", index, string(buf[:n]))

				}
			}
		}(&waitGroup)
		return writer
	}

	for i := 0; i < 10; i++ {
		waitGroup.Add(1)
		writer := readWriteFunc(i)
		go func(w *io.PipeWriter) {
			for {
				if doneWriting {
					_ = w.Close()
				}
			}
		}(writer)
		multiNWriter.AddWriter(i, writer)
	}

	err := multiNWriter.ShouldWrite([]byte("test1"))
	doneWriting = true
	if err != nil {
		t.Errorf("No writers should have failed")
	}
	waitGroup.Wait()
	close(resultsChannel)
	resultCount := 0
	for result := range resultsChannel {
		resultCount++
		fmt.Println(result)
	}
	if resultCount != 10 {
		t.Errorf("Expected 10 reads, got %d", resultCount)
	}
}
