#!/bin/sh

BENCH_OPTION='--workload 2'
START_WAIT=60

echo > /tmp/init_log.txt

(
echo '---start---'
date
echo '---env---'
env
echo '---cpuinfo---'
cat /proc/cpuinfo
echo '---ps---'
ps auxf

cd /home/isucon;

for i in {1..3}; do
  echo "---waiting $START_WAIT sec---"
  sleep $START_WAIT

  echo "---benchmarker ${i}---"
  /sbin/runuser -l isucon -c "./benchmarker bench $BENCH_OPTION" > /tmp/bench${i}.log
done

echo '---finished---'
date
) >> /tmp/init_log.txt 2>&1

