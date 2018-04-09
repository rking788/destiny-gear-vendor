#!/bin/bash 

# Crimson: 3437746471
# Garden Progeny 1: 472169727
# Positive Outlook: 3393130645
# Sunshot (messed up geometry for some reason)
hash="2907129557"
rm output/gear.scnassets/$hash.dae
rm output/gear.scnassets/*.jpg

## Texture-less
#go build && ./destiny-gear-vendor --cli --geom --dae --hash $hash

## With Textures
go build && ./destiny-gear-vendor --cli --geom --textures --dae --hash $hash
