IMAGE_NAME=aelse/xmppbot

xmppbot.linux-arm: *.go
	GOOS=linux GOARCH=arm go build -o xmppbot.linux-arm

docker-image: xmppbot.linux-arm
	docker build -t ${IMAGE_NAME}:latest .

clean:
	rm -f xmppbot.linux-arm xmppbot
