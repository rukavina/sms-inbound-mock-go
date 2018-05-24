#!/bin/bash

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

printf "\nStarting Test SMS Bot ...\n"
cd $BASEDIR/test_client
php -S localhost:9201 &

printf "\n\n\033[32;1m Inbound Mock UI at http://localhost:9200 \033[0m\n\n"

printf "\nStarting Inbound Mock Srv...\n"

cd $BASEDIR/
./sms-inbound-mock-go




