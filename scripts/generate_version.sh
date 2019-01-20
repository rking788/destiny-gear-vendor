#!/bin/sh

version=$(cat ../VERSION)
if [ -n "$CI_COMMIT_SHORT_SHA" ]
then
  hash="$CI_COMMIT_SHORT_SHA"
else
  hash="$(git rev-parse --short HEAD)"
fi

if [ -z $BUILD_NUMBER ]
then
    full_version="$version-$hash"
else
    full_version="$version-$hash-$BUILD_NUMBER"
fi

echo $full_version
