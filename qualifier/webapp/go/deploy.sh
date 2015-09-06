#!/bin/bash

set -ex

SSH_SERVER=instance-2.asia-east1-c.isucon5-1060
SSH_USER=isucon

echo "deploy start by $USER" | ../../notify_slack.sh
rsync -avz ../../ $SSH_USER@$SSH_SERVER:/home/isucon/isucon4/qualifier/
ssh -t $SSH_USER@$SSH_SERVER /bin/bash -c "/home/isucon/isucon4/qualifier/webapp/go/build_wrapper.sh"
ssh -t $SSH_USER@$SSH_SERVER pkill golang-webapp

echo "deploy finished $USER" | ../../notify_slack.sh
