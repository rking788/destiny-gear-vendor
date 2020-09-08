#!/bin/sh
docker run -it -v /Users/rking/dev/go/src/github.com/rking788/destiny-gear-vendor/docker-gear.scnassets:/root/output/gear.scnassets -e PORT=8181 -p 8181:8181 -p 55432:55432  -e DATABASE_URL=$DATABASE_URL -e BUNGIE_API_KEY=$BUNGIE_API_KEY registry.gitlab.com/rpk788/destiny-gear-vendor:1.0.0-baff123 ./server --cli --geom --textures --usdz --hash 821154603 

