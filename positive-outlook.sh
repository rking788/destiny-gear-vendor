#!/bin/bash 

# Crimson: 3437746471
# Garden Progeny 1: 472169727
# Positive Outlook: 3393130645
hash="3393130645"
rm output/gear.scnassets/$hash/$hash.dae
rm output/gear.scnassets/$hash/*.jpg

## Texture-less
#go build && ./destiny-gear-vendor --cli --geom --dae --hash $hash

## With Textures
#go build && ./destiny-gear-vendor --cli --geom --textures --dae --hash $hash

## With Textures USD format
go build && ./destiny-gear-vendor --cli --geom --textures --usd --hash $hash
