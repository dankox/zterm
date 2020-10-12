#!/bin/bash

for i in {1..5}
do
    echo "step " $i
    sleep 1s
done

echo "error step 6" >&2

for i in {7..10}
do
    echo "step " $i
    sleep 1s
done
