#!/bin/sh

if [ $# -eq 1 ]; then
  if [ "$1" == "--latest" ]; then
    latest="1"
  fi
fi

tag="$(cd ./scripts && sh generate_version.sh)"
echo "pushing tagged version"
docker push registry.gitlab.com/rpk788/destiny-gear-vendor:$tag

if [ "$latest" == "1" ]; then
  echo "Also requesting push latest"
  docker push registry.gitlab.com/rpk788/destiny-gear-vendor:latest
fi
