#!/bin/sh
docker run -it -e PORT=8181 -p 8181:8181 -e DATABASE_URL=$DATABASE_URL -e BUNGIE_API_KEY=$BUNGIE_API_KEY registry.gitlab.com/rpk788/destiny-gear-vendor:latest 
