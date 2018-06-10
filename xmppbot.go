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
		log.Printf("Received signal %v\n", sig)
		phs.Cancel()
	}()
}

type config struct {
	// Google Talk
	Server   string
	Username string
	Password string
	// Garage command
	MQTTAddr string
	MQTTUser string
	MQTTPass string
}

func parseFlags() (*config, error) {
	cfg := &config{}

	flag.StringVar(&cfg.Server, "server", "talk.google.com:443", "server")
	flag.StringVar(&cfg.Username, "username", "", "username")
	flag.StringVar(&cfg.Password, "password", "", "password")

	flag.StringVar(&cfg.MQTTAddr, "mqtt-addr", "", "garage MQTT server address")
	flag.StringVar(&cfg.MQTTUser, "mqtt-user", "", "garage mqtt username")
	flag.StringVar(&cfg.MQTTPass, "mqtt-pass", "", "garage mqtt password")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: xmppbot [options]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if cfg.Username == "" || cfg.Password == "" {
		flag.Usage()
		return nil, errors.New("missing username or password")
	}
	return cfg, nil
}

func main() {
	phs0 := phase.FromContext(context.Background())

	cfg, err := parseFlags()
	if err != nil {
		log.Printf("Could not get configuration: %v", err)
		os.Exit(1)
	}
	setupSignalHandler(phs0)

	phs1 := phs0.Next()
	go XMPPBot(phs1, cfg)

	<-phs0.Done()
}

func XMPPBot(phs phase.Phaser, cfg *config) {
	defer phs.Cancel()

	options := xmpp.Options{
		Host:     cfg.Server,
		User:     cfg.Username,
		Password: cfg.Password,
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
		"garage": garage(cfg.MQTTAddr, cfg.MQTTUser, cfg.MQTTPass),
		"ping":   ping,
	}

	for {
		chat, err := talk.Recv()
		if err != nil {
			log.Printf("Talk client error: %v\n", err)
			break
		}
		switch v := chat.(type) {
		case xmpp.Chat:
			log.Printf("%s: %s\n", v.Remote, v.Text)
			cmd := strings.Split(v.Text, " ")[0]
			if command, exists := commands[cmd]; exists {
				replyMsg := command(phs, v.Text)
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
			log.Printf("%s: %s\n", v.From, v.Show)
		}
	}
}
