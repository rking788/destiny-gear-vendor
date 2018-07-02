#!/bin/bash
## This script is meant to be run inside the docker container with the usd-toolset installed
pushd /home/root/workspace
export PYTHONPATH=$PYTHONPATH:/usr/local/USD/lib/python/

inputs=`find . -name \*.usda`
for i in $inputs; do
#    echo "Working on file $i"
#    echo "Changing to dir: $(dirname $i)"
    filename=$(basename $i)
    name="${filename%.*}"

    pushd $(dirname $i) > /dev/null
    echo "Outputting $name.usdc"
    /usr/local/USD/bin/usdcat -o "$name.usdc" $filename
    popd > /dev/null
done
