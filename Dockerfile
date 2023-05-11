FROM alpine:latest

# Create directories
WORKDIR /soft-serve
# Expose data volume
VOLUME /soft-serve

# Environment variables
ENV SOFT_SERVE_DATA_PATH "/soft-serve"
ENV SOFT_SERVE_INITIAL_ADMIN_KEYS ""

# Expose ports
# SSH
EXPOSE 23231/tcp
# HTTP
EXPOSE 23232/tcp
# Stats
EXPOSE 23233/tcp
# Git
EXPOSE 9418/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/soft", "serve" ]

RUN apk update && apk add --update git bash openssh && rm -rf /var/cache/apk/*

COPY soft /usr/local/bin/soft
