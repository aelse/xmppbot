# xmppbot

This bot connects to Google Talk (Hangouts) and listens for commands.
I wrote it to allow me to trigger opening a garage gate from a chat
session. It sends a message to the garage controller over MQTT.

## Usage

Build it, then run and provide the username and password for a google account.

    go build
    ./xmppbot -username <google_user> -password <google_pass> -mqtt-addr=... -mqtt-user=... -mqtt-pass=...

In a chat session running `ping` will trigger a response from the bot.

If the mqtt options are provided then it will also respond to the `garage` command.