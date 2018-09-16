#!/bin/bash 

# Crimson: 3437746471
# Garden Progeny 1: 472169727
# Positive Outlook: 3393130645
# Rat King Sparrow: 1173626681
# Rat King sidearm: 2362471601
# Jade Rabbit (Jester ornament)
# Whisper of the worm: 1891561814
# Sunshot (messed up geometry for some reason): 2907129557
#hash="2907129557"
#hash="1173626681"
hash="3393130645"
#hash="3437746471"
#hash="2362471601"
#hash="1970437989"
#hash="1891561814"
#rm output/gear.scnassets/$hash/$hash.dae
rm output/gear.scnassets/$hash/$hash.usda
rm output/gear.scnassets/$hash/$hash.usdc
rm output/gear.scnassets/$hash/*.jpg

## Texture-less
#go build && ./destiny-gear-vendor --cli --geom --dae --hash $hash

pushd cmd/server/
go build
popd

## With Textures
#./cmd/server/server --cli --geom --textures --dae --hash $hash

## With Textures in USD format
./cmd/server/server --cli --geom --textures --usd --hash $hash
