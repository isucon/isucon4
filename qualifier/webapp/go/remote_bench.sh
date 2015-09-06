#!/bin/bash

set -ex

SSH_SERVER=instance-2.asia-east1-c.isucon5-1060
SSH_USER=isucon

ssh -t $SSH_USER@$SSH_SERVER /home/isucon/isucon4/qualifier/webapp/go/bench.sh
