#!/bin/bash 

# Crimson: 3437746471
# Garden Progeny 1: 472169727
# Positive Outlook: 3393130645
# Rat King Sparrow: 1173626681
# Rat King sidearm: 2362471601
# Jade Rabbit (Jester ornament): 1970437989
# Whisper of the worm: 1891561814
# Sunshot (messed up geometry for some reason): 2907129557
# Ether doctor: 1839565992
# Trackless Waste: 2681395357
# Trinity Ghoul (exotic bow): 814876685
# Ace of Spades: 347366834
# Malfeasance (exotic hand cannon [thorn?]): 204878059
# Two-Tailed Fox (Exotic rocket): 2694576561
# One Thousand Voices: 2069224589
#hash="2907129557"
#hash="1173626681"
#hash="3393130645"
#hash="3437746471"
#hash="2362471601"
#hash="1970437989"
#hash="1891561814"
#hash="1839565992"
#hash="2681395357"
#hash="814876685"
#hash="347366834"
#hash="204878059"
#hash="2694576561"
hash="2069224589"
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
