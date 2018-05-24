#!/bin/bash

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

go build

cd $BASEDIR/test_client
composer install

cd $BASEDIR/public
bower install

cd $BASEDIR/
./start_dev.sh




