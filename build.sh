#!/bin/bash
#
# build.sh builds the application for deployment, putting everything into a
# `build` directory.
#

DEL="\033[0;34m==\033[00m"

echo -e "$DEL Generating CSS..."
make

echo -e "$DEL Building application..."
go build -i -x ./cmd/readas || { echo 'Build failed. Aborting.' ; exit 1; }

echo -e "$DEL Copying files..."
rm -rf build
mkdir build
cp -r keys/ build/
cp -r static/ build/
cp -r templates/ build/
cp cmd/readas/readas build/

echo -e "$DEL Done!"
