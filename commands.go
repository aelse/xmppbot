package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

type Command func(context.Context, string) string

func ping(ctx context.Context, msg string) string {
	return "pong"
}

func ip(ctx context.Context, msg string) string {
	var buf bytes.Buffer
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Sprintf("Couldn't get interfaces: %v", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			buf.Write([]byte("Could not get ip for interface "))
			buf.Write([]byte(iface.Name))
			buf.Write([]byte("\n"))
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			buf.Write([]byte(ip.String()))
			buf.Write([]byte("\n"))
		}
	}
	return buf.String()
}

func garage(addr, username, password string) Command {
	if addr == "" || username == "" || password == "" {
		return func(ctx context.Context, msg string) string {
			args := strings.Split(msg, " ")
			return args[0] + " is not configured."
		}
	}

	return func(ctx context.Context, msg string) string {
		args := strings.Split(msg, " ")
		usage := fmt.Sprintf("Usage: %s <entry|exit|ping>", args[0])

		if len(args) < 2 || (args[1] != "entry" && args[1] != "exit" && args[1] != "ping") {
			return usage
		}
		gate := args[1]
		topic := "garage/" + gate

		recvErr := make(chan error)

		mq := client.New(&client.Options{
			// Any error is pushed to the error channel, which triggers the failure path.
			ErrorHandler: func(err error) {
				recvErr <- err
			},
		})
		defer mq.Terminate()

		err := mq.Connect(&client.ConnectOptions{
			Network:  "tcp",
			Address:  addr,
			ClientID: []byte("optimus-prime"),
			UserName: []byte(username),
			Password: []byte(password),
		})
		if err != nil {
			return "Error connecting to MQTT server: " + err.Error()
		}
		defer mq.Disconnect()

		// Subscribe to topics.
		err = mq.Subscribe(&client.SubscribeOptions{
			SubReqs: []*client.SubReq{
				&client.SubReq{
					TopicFilter: []byte(topic),
					QoS:         mqtt.QoS1,
					// Define the processing of the message handler.
					Handler: func(topicName, message []byte) {
						// Saw message to our topic. Flag success by sending nil to error channel.
						recvErr <- nil
					},
				},
			},
		})
		if err != nil {
			return fmt.Sprintf("Error subscribing to topic %s: %v", topic, err.Error())
		}

		// Unsubscribe from topics upon return.
		defer func() {
			// Ignore error because we're closing the client anyway.
			_ = mq.Unsubscribe(&client.UnsubscribeOptions{
				TopicFilters: [][]byte{
					[]byte(topic),
				},
			})
		}()

		err = mq.Publish(&client.PublishOptions{
			QoS:       mqtt.QoS0,
			TopicName: []byte(topic),
			// The user triggering, in this case the bot.
			Message: []byte("optimus"),
		})
		if err != nil {
			return fmt.Sprintf("Error publishing to topic %s: %v", topic, err.Error())
		}

		select {
		// Channel returns an error when something failed or nil once a message seen on topic.
		case err = <-recvErr:
			if err != nil {
				return fmt.Sprintf("Failed to trigger gate: %v", err)
			}
			// Timeout after a few seconds of waiting.
		case <-time.NewTicker(5 * time.Second).C:
			return "Timed out waiting for message on topic."
		}

		if gate == "ping" {
			return "Pinged MQTT path successfully."
		}
		return "Opening the " + gate + " gate."
	}
}
