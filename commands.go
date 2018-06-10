package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

type Command func(context.Context, string) string

func ping(ctx context.Context, msg string) string {
	return "pong"
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
		usage := fmt.Sprintf("Usage: %s <entry|exit>", args[0])

		if len(args) < 2 || (args[1] != "entry" && args[1] != "exit") {
			return usage
		}
		topic := "garage/" + args[1]

		mq := client.New(&client.Options{
			// Define the processing of the error handler.
			ErrorHandler: func(err error) {
				fmt.Println(err)
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

		err = mq.Publish(&client.PublishOptions{
			QoS:       mqtt.QoS0,
			TopicName: []byte(topic),
			// The user triggering, in this case the bot.
			Message: []byte("optimus"),
		})
		if err != nil {
			return fmt.Sprintf("Error publishing to topic %s: %v", topic, err.Error())
		}

		return "Triggered opening the " + args[1] + " gate."
	}
}
