#!/bin/bash 

rm output/gear.scnassets/3437746471.dae

## Texture-less
go build && ./destiny-gear-vendor --cli --geom --dae --hash 3437746471

## With Textures
## go build && ./destiny-gear-vendor --cli --geom --textures --dae --hash 3437746471
