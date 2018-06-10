package main

import (
	"context"
	"errors"
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

type config struct {
	Server   string
	Username string
	Password string
}

func parseFlags() (*config, error) {
	cfg := &config{}
	flag.StringVar(&cfg.Server, "server", "talk.google.com:443", "server")
	flag.StringVar(&cfg.Username, "username", "", "username")
	flag.StringVar(&cfg.Password, "password", "", "password")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: example [options]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if cfg.Username == "" || cfg.Password == "" {
		flag.Usage()
		return nil, errors.New("invalid or missing args")
	}
	return cfg, nil
}

func main() {
	phs0 := phase.FromContext(context.Background())

	cfg, err := parseFlags()
	if err != nil {
		fmt.Printf("Could not get configuration: %v", err)
		os.Exit(1)
	}
	setupSignalHandler(phs0)

	phs1 := phs0.Next()
	go XMPPBot(phs1, cfg.Server, cfg.Username, cfg.Password)

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
