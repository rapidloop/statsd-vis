#!/bin/bash

set -eEo pipefail
trap 'exit 1' ERR

if [ "x$1" == "x" ]; then
	echo "Usage: scripts/build.sh VERSION"
	exit 1
fi
VERSION=$1

if [ ! -f 'main.go' ]; then
	echo "Run me from the root directory."
	exit 1
fi

function mkone {
	echo "Building for $GOOS/$GOARCH, version is $VERSION:"
	go build -o dist/statsd-vis-v$VERSION-$GOOS-$GOARCH *.go
	zip -9Xmj -o dist/statsd-vis-v$VERSION-$GOOS-$GOARCH.zip \
		dist/statsd-vis-v$VERSION-$GOOS-$GOARCH
}

rm -rf dist
mkdir dist

GOOS=linux   GOARCH=amd64 mkone
GOOS=windows GOARCH=amd64 mkone
GOOS=darwin  GOARCH=amd64 mkone

exit 0

