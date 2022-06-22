FROM alpine:latest

RUN apk update && apk add --update git && rm -rf /var/cache/apk/*

COPY soft /usr/local/bin/soft

# Create directories
WORKDIR /soft-serve
# Expose data volume
VOLUME /soft-serve

# Environment variables
ENV SOFT_SERVE_KEY_PATH "/soft-serve/ssh/soft_serve_server_ed25519"
ENV SOFT_SERVE_INITIAL_ADMIN_KEY ""
ENV SOFT_SERVE_REPO_PATH "/soft-serve/repos"

# Expose ports
# SSH
EXPOSE 23231/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/soft", "serve" ]
