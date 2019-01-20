#!/bin/bash

## Texture-less
#go build && ./destiny-gear-vendor --cli --geom --dae --hash $hash

## With Textures
#go build && ./destiny-gear-vendor --cli --geom --textures --dae --weapons

go build && ./destiny-gear-vendor --cli --geom --textures --usd --weapons
