#!/usr/bin/env bash

set -e
set -x


## if somebody does "docker run -it laminar <something>" let them...
echo "$@"
## so if there is no "-" default to laminar
if [ "${1:0:1}" = '-' ]; then

  ## prep SSH for ${GITHOST:-github.com}
  mkdir -p ~/.ssh && \
        ssh-keyscan -t rsa "${GITHOST:-github.com}" 2>/dev/null \
        > ~/.ssh/known_hosts

  chmod 0700 ~/.ssh

  eval "$(ssh-agent)"
  ssh-add "${SSH_KEY:-~/.ssh/id_rsa}"
	set -- /app/laminar "$@"
fi

## otherwise launch what the user asked for
exec "$@"
