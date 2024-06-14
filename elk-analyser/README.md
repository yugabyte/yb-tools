# Introduction

This tool is aimed to help Yugabye Support team to investigate customer's server log in a offline environment.

To achieve such goal, this tool utilize Filebeat to automatically upload the log files to a pre-built ELK cluster and enrich the log messages with pre-defined pipelines so that support engineer can easily read and search the log.

# ELK Setup 

Currently the tool only supports local docker ELK setup, please refer to the [official ELK docker setup guide](https://www.elastic.co/guide/en/elasticsearch/reference/current/docker.html) for reference.

Please also edit the `.env` and `docker-compose.yml` to meet your environment.

# First pipeline for tserver logs

Once the ELK cluster is up and running, Go to `Kibana UI` > `Dev Tools` and create the pipeline for tserver/master log pattern as below (Replace the pipeline name as your wish):

```
PUT _ingest/pipeline/<tserver_log>
{
"description": "testing grok processor",
    "processors": [
       {
            "grok": {
                "field": "message",
                "patterns": [
                "(?<loglevel>[IWEF])(?<timestamp>%{MONTHNUM}%{MONTHDAY} %{TIME})%{SPACE}%{NUMBER:threadid} %{NOTSPACE:filename}:%{NUMBER:fileline}%{NOTSPACE} %{GREEDYDATA:msg}"
                ]
            }
            },
            {
            "date": {
                "field": "timestamp",
                "formats": [
                "MMdd HH:mm:ss.SSSSSS"
                ],
                "target_field": "@timestamp"
            }
            },
            {
            "grok": {
                "field": "log.file.path",
                "patterns": [
                "^%{GREEDYDATA}/case/%{WORD:case_id}/%{GREEDYDATA}"
                ],
                "on_failure": [
                {
                    "append": {
                    "field": "case_id",
                    "value": [
                        "0"
                    ]
                    }
                }
                ]
            }
            },
            {
            "remove": {
                "field": [
                "timestamp",
                "log",
                "agent",
                "ecs",
                "host"
                ]
            }
            }
    ]
}
```

Application log pipeline is still WIP and will be updated once done.

# Filebeat Setup

Download and install Filebeat in your local environment following the [official guide](https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-installation-configuration.html).

Edit the `filebeat.yml.sample` file in the same folder, modify the `filebeat.inputs` and `output.elasticsearch` to match your environment.

Once done, rename the config file to `filebeat.yml` and run `filebeat -e -c filebeat.yml` to start the log consumption.
Note that depends on your platform the command can be slightly different, refer to the [official filebeat setup and run guide](https://www.elastic.co/guide/en/beats/filebeat/current/setting-up-and-running.html) for more details.

# Investigation

Go to `Kibana UI` > `Discover` and create a new View using `filebeat*` as `Index Pattern` and `@timestamp` as `Timestamp field`.

Now you can see the logs in this view.

# Rules & Alerts

It is often not clear what kind of issue is happening in customer's environment when we first start to investigate the issue.
With the help of ELK's rules and alert function, it is possible to automatically triage the logs with predefined rules and generate suggestions based on existing KBs.

To enable the rules & alerts, it is required to create an encryption key for kibana first.

Refer to the official [alert setup guide](https://www.elastic.co/guide/en/kibana/current/alerting-setup.html#alerting-prerequisites) and also .kibana.yml for more details.

# WIP

The next step of this tools is to create more pipelines to support master/application log pattern and also combine with Lincoln so that any tserver/master/application log uploaded to the Zendesk can be automatically parsed and kept in a centralized place.

Another goal of this tools is to create more alerts so that it will automatically generate notifications and suggest possible solutions (KB articles) based on parsed logs.

Also, the ELK setup section should also support standalone VM or K8S based installation.