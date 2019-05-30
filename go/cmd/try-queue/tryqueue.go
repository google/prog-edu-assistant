// Binary try-queue is for quick testing of the queue.
// It expects a default installation of RabbitMQ to be running
// on the default port (5672). The purpose of the tool
// is to make it possible for people who are new to message queues
// to play with the message queue and try it out for themselves.
//
// NOTE: This command is not needed for running production.
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
	Name string
	Help string
	Func func() error
}

var commands = []*Cmd{
	{"post", "Post a message to a queue.", postCommand},
	{"receive", "Receive a single message from a queue.", receiveCommand},
	{"listen", "Receive messages from a queue forever.", listenCommand},
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if *command == "" {
		fmt.Println("Error: --command not specified.")
		fmt.Println("Available commands:")
		for _, cmd := range commands {
			fmt.Printf("  %s  %s\n", cmd.Name, cmd.Help)
		}
		return nil
	}
	cmdMap := make(map[string]*Cmd)
	for _, cmd := range commands {
		cmdMap[cmd.Name] = cmd
	}
	cmd, ok := cmdMap[*command]
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
