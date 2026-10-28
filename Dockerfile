FROM alpine:latest

# Create directories
WORKDIR /soft-serve
# Expose data volume
VOLUME /soft-serve

# Environment variables
ENV SOFT_SERVE_DATA_PATH "/soft-serve"
ENV SOFT_SERVE_INITIAL_ADMIN_KEYS ""
# workaround to prevent slowness in docker when running with a tty
ENV CI "1"

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
ENTRYPOINT [ "/entrypoint.sh", "/usr/local/bin/soft", "serve" ]

RUN apk update && apk add --update git bash openssh su-exec && rm -rf /var/cache/apk/*

COPY entrypoint.sh /entrypoint.sh
COPY soft /usr/local/bin/soft
