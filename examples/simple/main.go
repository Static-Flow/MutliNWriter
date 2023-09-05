package main

import (
	"Static-Flow/MutliNWriter"
	"bufio"
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"io"
	"log"
	"os/exec"
	"sync"
)

func main() {
	//create the MultiNWriter
	multiWriter := MutliNWriter.NewMultiNWriter()

	//spawn an example task to listen to
	go func() {
		osCmd := exec.Command("/bin/bash", "-c", "while true; do echo 'data'; sleep 1; done")
		reader, writer := io.Pipe()
		bufferedReader := bufio.NewReader(reader)
		osCmd.Stdout = writer
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := osCmd.Run(); err != nil {
				log.Fatalln(err)
			}
			_ = writer.Close()
		}()

		buf := make([]byte, 1024)
		var n int
		var err error
		//read in task output and redirect it to the MultiNWriter
		for {
			n, err = bufferedReader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Println("Error reading from buffer:", err)
				return
			}
			err = multiWriter.Write(buf[:n])
			if err != nil {
				//example of how to unpack the errors from MultiNWriter.Write and handle them
				for _, multiWriterErr := range err.(interface{ Unwrap() []error }).Unwrap() {
					fmt.Printf("Writer error: %s on writer %s ", multiWriterErr.(MutliNWriter.WriteError).Error(), multiWriterErr.(MutliNWriter.WriteError).WriterKey())
					multiWriter.RemoveWriter(multiWriterErr.(MutliNWriter.WriteError).WriterKey())
				}

			}
		}
		wg.Wait()
	}()
	app := fiber.New()
	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		/*
			example way of connecting to the MultiNWriter to receive it's output. On websocket connect, a new
			child writer is added and its sibling read pipe is read from and passed to the websocket output.
		*/
		reader, writer := io.Pipe()
		multiWriter.AddWriter("key", writer)
		defer multiWriter.RemoveWriter("key")
		buf := make([]byte, 1024)
		for {
			_, err := reader.Read(buf)
			if err != nil {
				fmt.Println("read error: ", err)
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, buf)
			if err != nil {
				fmt.Println("web socket write error: ", err)
				return
			}
		}
	}))
	log.Fatal(app.Listen(":8080"))
}
