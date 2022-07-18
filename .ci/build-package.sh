#!/bin/bash

set -x
set -e

cp mielesolar synology/bin
cd synology
./INFO.sh > INFO
tar cvfz package.tgz bin
tar -c -v --exclude INFO.sh --exclude repo --exclude mielesolar*.spk \
  -f mielesolar-"${SPK_ARCH:-x86_64}"-"${SPK_PACKAGE_SUFFIX:-latest}".spk \
  ./*
