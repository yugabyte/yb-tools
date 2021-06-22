#!/bin/bash

set -x

ARGS=$(find . -iname "*.proto" -printf "%P\n" | perl -lane '/(.*)\/([^\/]*)\/([^\/]*).proto/; print "--go_opt=M$_=github.com/yugabyte/yb-tools/yugatool/api/$1/$2;$2 --ybrpc_opt=M$_=github.com/yugabyte/yb-tools/yugatool/api/$1/$2;$2"')


mkdir -p out
find yb -iname *.proto | xargs protoc $ARGS --go_out=./../api/ --ybrpc_out=./../api/
