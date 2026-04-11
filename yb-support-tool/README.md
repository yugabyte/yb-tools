# yb-support-tool

`yb-support-tool` is a collection of tools to ease interaction with Yugabyte Support.

- `upload` -> allows direct upload of files to support tickets
- `exec` -> allows commands to be executed across all nodes in a universe (or multiple universes)
- `yugaware-client` -> bundles the existing `yugaware-client` from `yb-tools`
- `yugatool` -> bundles the existing `yugatool` from `yb-tools`

<br>

```
$ ./yb-support-tool
Yugabyte Support Tool

Usage:
  yb-support-tool [command]

Available Commands:
  completion      Generate the autocompletion script for the specified shell
  exec            Run commands across all nodes in a universe
  help            Help about any command
  upload          Upload attachment to a support case
  version         
  yugatool        A tool to make troubleshooting yugabyte somewhat easier
  yugaware-client A tool to deploy/configure yugaware 

Flags:
  -h, --help      help for yb-support-tool
      --verbose   Print verbose logs

Use "yb-support-tool [command] --help" for more information about a command.
```


## upload

The upload command uploads one or more files to a support ticket. It can also be used to provide files to a custom dropzone with the `--dropzone_id` flag.

There is a limit of 10 files and aggregate file size of 100 GB, so files larger than 100 GB must be compressed or split before uploading. This is not enforced, but the upload will fail.

```
Usage:
  yb-support-tool upload -c [case number] -e [email] [files] [flags]

Flags:
  -c, --case int             Zendesk case number to attach files to (required)
      --dropzone_id string   Override default dropzone ID (default "BdFZz_JoZqtqPVueANkspD86KZ_PJsW1kIf_jVHeCO0")
  -e, --email string         Email address of submitter (required)
  -h, --help                 help for upload
      --retries uint         number file upload retry attempts (default 5)

Global Flags:
      --verbose   Print verbose logs
```

## exec

The exec command is used to execute commands over all nodes in a universe (or multiple universes). Universe details can be provided in two ways:

- using the "yba" command, which will invoke `yugaware-client` to programmatically query the YBA API.

- using the "file" command, and manually providing a `json` or `yaml
inventory file.


```
Usage:
  yb-support-tool exec [flags]
  yb-support-tool exec [command]

Available Commands:
  file        Lookup inventory using inventory file
  yba         Lookup inventory using YBA API

Flags:
      --cmd string        command to run across all universe nodes
  -h, --help              help for exec
  -p, --parallel          run commands against all nodes simultaneously
      --universe string   universe to run commands against
  -u, --user string       ssh user to run commands with (default "yugabyte")
      --verbose           show node details next to ssh output

Use "yb-support-tool exec [command] --help" for more information about a command.
```

#### inventory file templates

```json
[
    {
       "name":"yba-replicated-universe-1",
       "universeDetails":{
          "clusters":[
             {
                "clusterType":"PRIMARY",
                "userIntent":{
                   "accessKeyCode":"yb-dev-<snip>-key",
                   "provider":"<provider_uuid>"
                }
             }
          ],
          "nodeDetailsSet":[
             {
                "cloudInfo":{
                   "private_ip":"10.10.10.100"
                },
                "nodeName":"yb-dev-universe-1-n2",
                "nodeIdx":"2"
             },
             {
                "cloudInfo":{
                   "private_ip":"10.10.10.101"
                },
                "nodeName":"yb-dev-universe-1-n3",
                "nodeIdx":"3"
             },
             {
                "cloudInfo":{
                   "private_ip":"10.10.10.102"
                },
                "nodeName":"yb-dev-universe-1-n1",
                "nodeIdx":"1"
             }
          ],
          "nodePrefix":"yb-dev-universe-1"
       }
    }
 ]   
 ```

```yaml
---
- name: yba-replicated-universe-1
  universeDetails:
    clusters:
    - clusterType: PRIMARY
      userIntent:
        accessKeyCode: yb-dev-<snip>-key
        provider: <provider_uuid>
    nodeDetailsSet:
    - cloudInfo:
        private_ip: 10.10.10.100
      nodeName: yb-dev-universe-1-n2
      nodeIdx: '2'
    - cloudInfo:
        private_ip: 10.10.10.101
      nodeName: yb-dev-universe-1-n3
      nodeIdx: '3'
    - cloudInfo:
        private_ip: 10.10.10.102
      nodeName: yb-dev-universe-1-n1
      nodeIdx: '1'
    nodePrefix: yb-dev-universe-1
```


## `yugatool` and `yugaware-client`

Running `./yb-support-tool yugatool` and `./yb-support-tool yugaware` will run the existing utilities from `yb-tools`

```
./yb-support-tool yugatool 
A tool to make troubleshooting yugabyte somewhat easier

Usage:
  yb-support-tool yugatool [command]

Available Commands:
  cluster_info Export cluster information
  healthcheck  Run yugabyte health checks
  tablet_info  Get tablet consensus info and state
  util         Miscellaneous utilities
  xcluster     Various utilities to interract with xcluster replication

Flags:
  -c, --cacert string             the path to the CA certificate
      --client-cert string        the path to the client certificate
      --client-key string         the path to the client key file
      --config string             config file (default is $HOME/.yugatool.yaml)
      --debug                     debug mode
      --dial-timeout int          number of seconds for dial timeouts (default 10)
  -h, --help                      help for yugatool
  -m, --master-addresses string   comma-separated list of YB Master server addresses (minimum of one)
  -o, --output string             Output options as one of: [table, json, yaml] (default "table")
      --skiphostverification      skip tls host verification
  -v, --version                   version for yugatool

Global Flags:
      --verbose   Print verbose logs

Use "yb-support-tool yugatool [command] --help" for more information about a command.
```

```
$ ./yb-support-tool yugaware-client
A tool to deploy/configure yugaware

Usage:
  yb-support-tool yugaware-client [command]

Available Commands:
  backup      Interact with Yugabyte backups
  certificate Interact with Yugabyte certificates
  login       Login to a Yugaware server
  provider    Interact with Yugaware providers
  register    Register a Yugaware server
  session     Session management utilities
  storage     Interact with Yugaware storage
  universe    Interact with Yugabyte universes
  version     Print the client and server version

Flags:
      --api-token string       api token for yugaware session
  -c, --cacert string          the path to the CA certificate
      --client-cert string     the path to the client certificate
      --client-key string      the path to the client key file
      --config string          config file (default is $HOME/.yugatool.yaml)
      --debug                  debug mode
      --dialtimeout int        number of seconds for dial timeouts (default 10)
  -h, --help                   help for yugaware-client
      --hostname string        hostname of yugaware (default "localhost:8080")
  -o, --output string          Output options as one of: [table, json, yaml] (default "table")
      --skiphostverification   skip tls host verification

Global Flags:
      --verbose   Print verbose logs
```

## TODO

- Auto split and package files greater than 100GB in size
