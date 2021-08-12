FROM alpine:latest

RUN apk update && apk add --update nfs-utils git && rm -rf /var/cache/apk/*

COPY smoothie /usr/local/bin/smoothie

# Create directories
WORKDIR /smoothie
# Expose data volume
VOLUME /smoothie

# Environment variables
ENV SMOOTHIE_KEY_PATH "/smoothie/ssh/smoothie_server_ed25519"
ENV SMOOTHIE_REPO_KEYS_PATH "/smoothie/ssh/smoothie_git_authorized_keys"
ENV SMOOTHIE_REPO_PATH "/smoothie/repos"

# Expose ports
# SSH
EXPOSE 23231/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/smoothie" ]