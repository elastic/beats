#!/bin/sh

#
# Works around the fact that go tool doesn't support the recursive
# flag when using the coverage flag.
# Taken from here: https://gist.github.com/hailiang/0f22736320abe6be71ce
# Called from the Makefile.

echo "mode: count" > profile.cov

# Standard go tooling behavior is to ignore dirs with leading underscors
for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -type d);
do
if ls $dir/*.go &> /dev/null; then
    echo $dir
    go test -covermode=count -coverprofile=$dir/profile.tmp $dir
    if [ -f $dir/profile.tmp ]
    then
        cat $dir/profile.tmp | tail -n +2 >> profile.cov
        rm $dir/profile.tmp
    fi
fi
done
