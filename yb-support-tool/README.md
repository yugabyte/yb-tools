# yb-support-tool

`yb-support-tool` is a collection of tools to ease interaction with Yugabyte Support.

Today, the tool supports an "upload" command which enables files to be uploaded to a provided support ticket


## upload

The upload command uploads one or more files to a support ticket. It can also be used to provide files to a custom dropzone with the `--dropzone_id` flag.

There is a limit of 10 files and aggregate file size of 100 GB, so files larger than 100 GB must be compressed or split before uploading. This is not enforced, but the upload will fail.



## TODO

- Auto split and package files greater than 100GB in size
