#!/bin/bash

## Clear out the old usdz cache
#rm "output/usdz-cache/*.usdz"

## Use the usd-toolset in Linux to convert USDA -> USDC
docker run -v $(pwd):/home/root/workspace usd-toolset "/home/root/workspace/convert-usd.sh"

## Create all USDZ files from USDC and textures

inputs=`find . -name \*.usdc`
for i in $inputs; do
    filename=$(basename $i)
    name="${filename%.*}"

    pushd $(dirname $i) > /dev/null
    echo "Outputting $name-unaligned.usdz"

    ## Create an uncompressed zip with only the .usdc file, this way it is always the first entry
    zip -0 -q "$name-unaligned.usdz" $filename

    ## Update the same zip with the texture files, ensuring they come AFTER the usdc file. very important
    zip -0 -q -u "$name-unaligned.usdz" *.jpeg *.jpg *.png

    ## zipalign, need to install Android dev tools to get this
    ## USDZ files are required to be 64 byte aligned, I think zipalign edits the zip header data
    echo "Zipalign-ing usdz..."
    zipalign -f 64 "$name-unaligned.usdz" "$name.usdz"

    rm "$name-unaligned.usdz"

    ## Add this new usdz file to the usdz cache directory
    #mv "$name.usdz" ../../usdz-cache/

    popd > /dev/null
done
