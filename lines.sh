#!/bin/bash

for f in `git ls-files | grep -v vendor | grep '.go'`
do
    echo "checking $f"
    lines=`awk '{if ( length($0) > 80 ) { print length($0) }}' "$f"`
    for l in $lines
    do
        echo "$l $f"
    done
done
