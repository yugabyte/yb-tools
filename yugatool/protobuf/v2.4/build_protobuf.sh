#!/bin/bash

set -x

ARGS=$(find . -iname "*.proto" -printf "%P\n" | perl -lane '/(.*)\/([^\/]*)\/([^\/]*).proto/; print "--go_opt=M$_=github.com/yugabyte/yb-tools/yugatool/api/v2.4/$1/$2;$2"')


mkdir -p out
find yb -iname *.proto | xargs protoc $ARGS --go_out=./../../api/v2.4/
