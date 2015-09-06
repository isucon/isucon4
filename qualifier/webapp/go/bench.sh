#!/bin/bash

setlock -Xn /tmp/benchmark.lock /home/isucon/benchmarker bench $* | /home/isucon/notify_slack
