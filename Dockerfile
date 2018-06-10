FROM armhf/alpine

RUN apk update && apk add ca-certificates

ADD xmppbot.linux-arm /usr/bin/xmppbot

CMD /usr/bin/xmppbot -username=${GOOGLE_USER} -password=${GOOGLE_PASS} -mqtt-addr=${MQTT_ADDR} -mqtt-user=${MQTT_USER} -mqtt-pass=${MQTT_PASS}
