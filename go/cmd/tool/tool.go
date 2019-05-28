// Binary tools is for quick testing of the queue.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/google/prog-edu-assistant/queue"
)

var (
	spec      = flag.String("spec", "amqp://guest:guest@localhost:5672/", "The spec of the queue to connect to.")
	queueName = flag.String("queue", "", "The name of the queue.")
	command   = flag.String("command", "", "The command to perform.")
	message   = flag.String("message", "", "The message to send. If not specified, --file is used.")
	file      = flag.String("file", "", "The path to the file to send.")
)

type Cmd struct {
	Help string
	Func func() error
}

var commands = map[string]*Cmd{
	"post":    &Cmd{"Post a message to a queue.", postCommand},
	"receive": &Cmd{"Receive a single message from a queue.", receiveCommand},
	"listen":  &Cmd{"Receive messages from a queue forever.", listenCommand},
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if *command == "" {
		fmt.Println("Error: --command not specified.")
		fmt.Println("Available commands:")
		for name, cmd := range commands {
			fmt.Printf("  %s  %s\n", name, cmd.Help)
		}
		return nil
	}
	cmd, ok := commands[*command]
	if !ok {
		return fmt.Errorf("unknown command: %q", *command)
	}
	return cmd.Func()
}

var channel *queue.Channel

func initQueue() error {
	var err error
	channel, err = queue.Open(*spec)
	return err
}

func postCommand() error {
	err := initQueue()
	if err != nil {
		return err
	}
	return channel.Post(*queueName, []byte(*message))
}

func receiveCommand() error {
	err := initQueue()
	if err != nil {
		return err
	}
	ch, err := channel.Receive(*queueName)
	if err != nil {
		return err
	}
	b := <-ch
	fmt.Printf("Received: %q\n", string(b))
	return nil
}

func listenCommand() error {
	err := initQueue()
	if err != nil {
		return err
	}
	ch, err := channel.Receive(*queueName)
	if err != nil {
		return err
	}
	for b := range ch {
		fmt.Printf("Received: %q\n", string(b))
	}
	return nil
}
