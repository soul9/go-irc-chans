#!/bin/bash

PIDS=""

testii() {
     cd "/tmp/irctests/viotest.local" || return 1
     test -d "$1" || echo "/j $1" > in
     sleep 5
     cd "$1" || return 1
     cat /home/johnny/dev/go/go-markov/corpuss/anar.txt | while builtin read -r line; do
           echo $line > in || return 1
           sleep 0.1
      done
}

rm -r /tmp/irctests/viotest.local/
ii -s viotest.local -i "/tmp/irctests/" -n "ii" -f "ii" &
PIDS="$PIDS $!"
sleep 15

for i in $(seq 15); do
    (while testii "#soul9"; do sleep 1; done) &
    PIDS="$PIDS $!"
done

for i in $(seq 16 30); do
    (while testii "#ubuntu"; do sleep 1; done) &
    PIDS="$PIDS $!"
done

echo $PIDS
read
kill -9 $PIDS || echo "Problem killing subprocesses, trying again"
for pid in $PIDS; do
    kill $pid || kill -9 $pid || echo "Couldn't kill $pid"
done
#be really sure
ps axuf |grep $0 |awk '{print $2}' |tr '\n' ' ' |xargs kill
killall ii
rm -r /tmp/irctests/viotest.local/
