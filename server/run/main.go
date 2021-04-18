package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mal-as/chat/server"
	"github.com/mal-as/chat/server/storage/mem"
)

func main() {
	db := mem.NewStorage()
	srv, err := server.New(":7777", db)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	srv.Start(ctx)
	log.Println("Start")

	notify := make(chan os.Signal, 1)
	signal.Notify(notify, os.Interrupt, syscall.SIGTERM)

	<-notify

	log.Println("Stopping...")

	srv.Stop(cancel)

	log.Println("Stopped")
}
