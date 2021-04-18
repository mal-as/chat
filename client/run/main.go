package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mal-as/chat/client"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	clnt, err := client.New("localhost:7777")
	if err != nil {
		log.Fatal(err)
	}

	clnt.Start(ctx)

	notify := make(chan os.Signal, 1)
	signal.Notify(notify, os.Interrupt, syscall.SIGTERM)

	<-notify

	clnt.Stop(cancel)

	log.Println("Stopped")
}
