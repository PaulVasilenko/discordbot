FROM alpine:3.8
RUN mkdir -p /etc/discordbot/
COPY discordbot chrisify /usr/bin/
COPY config.conf /etc/discordbot/

RUN apk add --no-cache ca-certificates

RUN mkdir -p /home/faces
COPY faces /home/faces

ENTRYPOINT ["/usr/bin/discordbot", "-c", "/etc/discordbot/config.conf"]