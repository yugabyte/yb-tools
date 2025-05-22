#!/bin/bash

#set -x

ARGS=$(find . -iname "*.proto" -printf "%P\n" | perl -lane '/(.*)\/([^\/]*)\/([^\/]*).proto/; print "--go_opt=M$_=github.com/yugabyte/yb-tools/yugatool/api/$1/$2;$2 --ybrpc_opt=M$_=/$1/$2;$2"')

for protofile in $(find yb yugatool -iname *.proto); do
  echo "Compiling $protofile"
  protoc ${ARGS} --go_out=./../api/ --ybrpc_out=./../api/ --go_opt=paths=source_relative $protofile
done