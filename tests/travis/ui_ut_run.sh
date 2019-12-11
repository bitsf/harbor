#!/bin/bash
set -x
set -e

cd ./src/portal
sudo npm install -q -g --no-progress --unsafe-perm=true --allow-root angular-cli
sudo npm install -q -g --no-progress --unsafe-perm=true --allow-root karma
npm install -q --no-progress
npm run test && cd -