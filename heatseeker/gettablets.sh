#!/bin/sh
# fetch tablet directories

if [ "$1" == "" ]; then
        echo USAGE: $0 host [host...]
        exit
fi

for host in $*
do
	echo $host
	ssh -t -i $HOME/pem -ostricthostkeychecking=no yugabyte@${host} "find /mnt/d0/yb-data/tserver/data/rocksdb -type f -exec stat -c '%s %n' '{}' \;" > data.$host &
done

wait
