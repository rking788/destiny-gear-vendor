#!/bin/sh

## This script is meant to be run inside the docker container and will
## build all required USD tools and cleanup afterwards. The goal is to have
## a (significantly) smaller resulting docker image without the intermediate files
## generated during the build process

python USD/build_scripts/build_usd.py /usr/local/USD

rm -rf /usr/local/USD/build /usr/local/USD/src /usr/local/USD/include /usr/local/USD/share /root/USD
