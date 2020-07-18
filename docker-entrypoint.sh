#!/usr/bin/env bash


set -e
set -x


## if somebody does "docker run -it laminar <something>" let them...

echo "$@"
## so if there is no "-" default to laminar
if [ "${1:0:1}" = '-' ]; then
  eval `ssh-agent`
  ssh-add ${SSH_KEY}
	set -- /app/laminar $@
fi

## otherwise launch what the user asked for
exec "$@"
