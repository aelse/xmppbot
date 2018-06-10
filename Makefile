IMAGE_NAME=aelse/xmppbot

xmppbot.linux-arm:
	GOOS=linux GOARCH=arm go build -o xmppbot.linux-arm

docker-image: xmppbot.linux-arm
	docker build -t ${IMAGE_NAME}:latest .
