package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aelse/phase"
	"github.com/mattn/go-xmpp"
)

func setupSignalHandler(phs phase.Phaser) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		phs.Cancel()
	}()
}

func main() {
	phs0 := phase.FromContext(context.Background())

	var server = flag.String("server", "talk.google.com:443", "server")
	var username = flag.String("username", "", "username")
	var password = flag.String("password", "", "password")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: example [options]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	if *username == "" || *password == "" {
		flag.Usage()
	}

	setupSignalHandler(phs0)

	phs1 := phs0.Next()
	go XMPPBot(phs1, *server, *username, *password)

	<-phs0.Done()
}

func XMPPBot(phs phase.Phaser, addr, username, password string) {
	defer phs.Cancel()

	options := xmpp.Options{
		Host:     addr,
		User:     username,
		Password: password,
	}

	talk, err := options.NewClient()
	if err != nil {
		log.Printf("Could not create talk client: %v\n", err)
		return
	}

	// Close everything when our context ends.
	go func() {
		<-phs.Done()
		talk.Close()
		phs.Cancel()
	}()

	commands := map[string]Command{
		"ping": ping,
	}

	for {
		chat, err := talk.Recv()
		if err != nil {
			log.Printf("Talk client error: %v\n", err)
			break
		}
		switch v := chat.(type) {
		case xmpp.Chat:
			fmt.Println(v.Remote, v.Text)
			cmd := strings.Split(v.Text, " ")[0]
			if command, exists := commands[cmd]; exists {
				replyMsg := command(v.Text)
				reply := xmpp.Chat{
					Remote: v.Remote,
					Type:   "chat",
					Text:   replyMsg,
				}
				if _, err := talk.Send(reply); err != nil {
					log.Printf("Failed to send reply: %v", err)
				}
			}
		case xmpp.Presence:
			fmt.Println(v.From, v.Show)
		}
	}
}
