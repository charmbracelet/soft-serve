FROM alpine:latest

RUN apk update && apk add --update nfs-utils git && rm -rf /var/cache/apk/*

COPY soft-serve /usr/local/bin/soft-serve

# Create directories
WORKDIR /soft-serve
# Expose data volume
VOLUME /soft-serve

# Environment variables
ENV SOFT_SERVE_KEY_PATH "/soft-serve/ssh/soft_serve_server_ed25519"
ENV SOFT_SERVE_REPO_KEYS ""
ENV SOFT_SERVE_REPO_KEYS_PATH "/soft-serve/ssh/soft_serve_git_authorized_keys"
ENV SOFT_SERVE_REPO_PATH "/soft-serve/repos"

# Expose ports
# SSH
EXPOSE 23231/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/soft-serve" ]