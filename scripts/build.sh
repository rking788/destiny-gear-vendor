#!/bin/sh

if [ $# -eq 1 ]; then
  if [ "$1" == "--latest" ]; then
    latest="1"
  fi
fi

if [ "$latest" == "1" ]; then
  echo "Also requesting build latest"
  tags="-t registry.gitlab.com/rpk788/destiny-gear-vendor:$(cd ./scripts && sh generate_version.sh) -t registry.gitlab.com/rpk788/destiny-gear-vendor:latest"
else
  tags="-t registry.gitlab.com/rpk788/destiny-gear-vendor:$(cd ./scripts && sh generate_version.sh)"
fi

echo "Building tags: $tags"
docker build $tags .

#docker run -it -e PORT=8181 -p 8181:8181 -e DATABASE_URL=$DATABASE_URL -e BUNGIE_API_KEY=$BUNGIE_API_KEY registry.gitlab.com/rpk788/destiny-gear-vendor:$(cd ./scripts && bash generate_version.sh)
