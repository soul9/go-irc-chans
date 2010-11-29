#!/bin/bash

PIDS=""

testii() {
     cd "/tmp/irctests/$2/viotest.local" || return 1
     test -d "$1" || echo "/j $1" > in
     sleep 5
     cd "$1" || return 1
     cat $3 | while builtin read -r line; do
           echo $line > in || return 1
           sleep 0.1
      done
}

rm -r /tmp/irctests/viotest.local/
ii -s viotest.local -i "/tmp/irctests/1" -n "ii1" -f "ii" &
ii -s viotest.local -i "/tmp/irctests/2" -n "ii2" -f "ii" &
IIPIDS="$IIPIDS $!"
sleep 15
for i in /tmp/irctests/1 /tmp/irctests/2; do
    echo "/j #ubuntu" > $i/viotest.local/in
    sleep 1
    echo "/j #soul9" > $i/viotest.local/in
done
sleep 3

dotests() {
    for i in $(seq 15); do
        (while testii "#soul9" 1 /home/johnny/dev/go/go-markov/corpuss/anar.txt; do sleep 1; done) &
        PIDS="$PIDS $!"
    done
    for i in $(seq 15); do
        (while testii "#soul9" 2 /home/johnny/dev/go/go-markov/corpuss/king_james.txt; do sleep 1; done) &
        PIDS="$PIDS $!"
    done

    for i in $(seq 16 30); do
        (while testii "#ubuntu" 1 /home/johnny/dev/go/go-markov/corpuss/anar.txt; do sleep 1; done) &
        PIDS="$PIDS $!"
    done
    for i in $(seq 16 30); do
        (while testii "#ubuntu" 2 /home/johnny/dev/go/go-markov/corpuss/king_james.txt; do sleep 1; done) &
        PIDS="$PIDS $!"
    done
}

killtests() {
    kill -9 $PIDS || echo "Problem killing subprocesses, trying again"
    for pid in $PIDS; do
        kill $pid || kill -9 $pid || echo "Couldn't kill $pid"
    done
}

dotests
echo $PIDS
read
killtests

#be really sure
kill $IIPIDS
killall ii || killall -9 ii
ps axuf |grep $0 |awk '{print $2}' |tr '\n' ' ' |xargs kill
rm -r /tmp/irctests/
