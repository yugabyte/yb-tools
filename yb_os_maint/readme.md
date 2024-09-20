# yb_os_maint.py instructions

This script is used to prepare for node maintenance such as O/S patching.  It is intended to run on the node to be shut down and can be placed there using automation tools such as Ansible/SALT.  The script will do the following:
 - Perform a healthcheck on one or more universe(s) including the following:
   -  Check for dead nodes, T-Server and Masters
   -  Check master and tablet health
   -  Ensure all nodes have been up for the specified amount of time
   -  Ensure tablets are balanced
   -  Check both tablet and master lag to ensure they are below the specified threshold
 - Stop or resume a node:
   - For YBA hosts, will stop/start the YBA related services on the node it is run on
   - For DB nodes:
     - Pause x-cluster replication if stopping a single node and not skipping
     - Stop T-Server and Master processes
     - Create a maitenance window to stop alerts if stopping a single node and not skipping
     - Resume is the reverse of the above
  - Reprovision a node
# Usage
```
usage: yb_os_maint.py [-h]
                      (-s | -p | -r | -t [HEALTH] | -f FIX [FIX ...] | -v)
                      [-d] [-l LOG_FILE_DIRECTORY] [-x] [-m] [-g REGION]
                      [-a AVAILABILITY_ZONE] [-e ENV_FILE_PATH] [-u UNIVERSE]
                      [-k]

Yugabyte pre/post flight check - Start and Stop Services before and after O/S
patch

optional arguments:
  -h, --help            show this help message and exit
  -s, --stop            Stop services for YB host prior to O/S patch
  -p, --reprovision     Re-Provision (dead) node before bringing it back to
                        life
  -r, --resume, --start
                        Resume services for YB host after O/S patch
  -t [HEALTH], --health [HEALTH]
                        Healthcheck - specify Universe Name or "ALL" if not
                        running on a DB Node
  -f FIX [FIX ...], --fix FIX [FIX ...]
                        Fix one or more of the following: placement
  -v, --verify          Verify Master and tServer process are in correct state
                        per universe config
  -d, --dryrun          Dry Run - health checks only - no actions taken.
  -l LOG_FILE_DIRECTORY, --log_file_directory LOG_FILE_DIRECTORY
                        Log file folder location. Output is sent to terminal
                        stdout if not specified.
  -x, --skip_xcluster   Skip Pause or Resume of xCluster replication when
                        stopping or resuming nodes - False if not specified,
                        forced to True if stopping multiple nodes in a
                        region/AZ.
  -m, --skip_maint_window
                        Skip creation/removal of maintenence window when
                        stopping or resuming nodes - False if not specified,
                        forced to True if stopping multiple nodes in a
                        region/AZ.
  -g REGION, --region REGION
                        Region for nodes to be stopped/resumed - action taken
                        on local node if not specified.
  -a AVAILABILITY_ZONE, --availability_zone AVAILABILITY_ZONE
                        AZ for nodes to be stopped/resumed(optional) - Script
                        will abort if --region is not specified along with the
                        AZ
  -e ENV_FILE_PATH, --ENV_FILE_PATH ENV_FILE_PATH
                        By default, the script will look for the
                        ENV_FILE_PATTERN in /home/yugabyte. You can overwrite
                        this by providing --ENV_FILE_PATH <new path>, example
                        is --ENV_FILE_PATH /tmp/
  -u UNIVERSE, --universe UNIVERSE
                        Universe to operate on, or "ALL" (health, or regional
                        ops)
  -k, --skip_stepdown   Skip master-stepdown if this is a STOP on a master-
                        leader. If not set, we will attempt stepdown.
```
                        
# Setup
1. Tune the parameters in the file (see the variables under '##Globals')
2. Script should be placed on the server to be shutdown/resumed.  It will detect if the server is a YBA or DB node.
3. Create a file with the name pattern '.yba_&lt;appname&gt;.rc' for the YBA information.  This should include the following exports:
    -  export YBA_HOST='&lt;YBA Host URL&gt;'
    -  export API_TOKEN='&lt;API Token&gt;'
    -  export CUST_UUID='&lt;Customer ID&gt;'
  
   These values can also be supplied directly via environment variables (no need for the .rc file)

# Sample output for stop command:
```
--------------------------------------
2024-10-25 20:37:51 script started
Retrieving environment variables from file /home/yugabyte/.yba_dev.rc
Checking host yb-dev-xcluster-src-n1 with IP 10.202.0.54
Checking if host 10.202.0.54 is YBA Instance...
  Host is NOT YBA Instance - checking if it is a DB Node
Checking current node using YBA server at http://10.138.15.193
Performing pre-flight checks...

Checking for prometheus host at http://10.138.15.193:9090/api/v1/query
Using prometheus host at http://10.138.15.193:9090/api/v1/query

Found node yb-dev-xcluster-src-n1 in Universe xcluster-src - UUID d79fb004-ab09-4fff-a2b8-b1a06195b68b
- Checking for task placement UUID
  Passed placement task test
- Checking for dead nodes, tservers and masters
  All nodes, masters and t-servers are alive.
- Checking for master and tablet health
  Most recent uptime: 1353 seconds
  No under replicated or leaderless tablets found
- Checking tablet health
  --- Tablet Report --
  TServer 10.202.0.97:9000 Active tablets: 36, User tablets: 24, User tablet leaders: 0, System Tablets: 12, System tablet leaders: 0
  TServer 10.202.0.54:9000 Active tablets: 36, User tablets: 24, User tablet leaders: 12, System Tablets: 12, System tablet leaders: 6
  TServer 10.202.0.55:9000 Active tablets: 36, User tablets: 24, User tablet leaders: 12, System Tablets: 12, System tablet leaders: 6
  TServer 10.215.0.36:9000 Active tablets: 18, User tablets: 12, User tablet leaders: 0, System Tablets: 6, System tablet leaders: 0
  TServer 10.215.0.19:9000 Active tablets: 18, User tablets: 12, User tablet leaders: 0, System Tablets: 6, System tablet leaders: 0
  Passed tablet balance check - YBA is reporting the following: Cluster Load is Balanced
  Checking replication lag for t-servers
  Executing the following Prometheus query:
   max(max_over_time(follower_lag_ms{exported_instance=~"yb-dev-xcluster-src-n1|yb-dev-xcluster-src-n5|yb-dev-xcluster-src-readonly1-n1|yb-dev-xcluster-src-n3|yb-dev-xcluster-src-n2|",export_type="tserver_export"}[10m]))
  Check passed - t-server lag of 0.638 seconds is below threshhold of 60
- Checking  master health
  Checking for underreplicated masters
  Check passed - cluster has RF of 3 and 3 masters
  Checking replication lag for masters
  Executing the following Prometheus query:
   max(max_over_time(follower_lag_ms{exported_instance=~"yb-dev-xcluster-src-n1|yb-dev-xcluster-src-n3|yb-dev-xcluster-src-n2|",export_type="master_export"}[10m]))
  Check passed - master lag of 0.6 seconds is below threshhold of 60
--- Pre flight check completed with no issues

- Creating Maintenance window "OS Patching - yb-dev-xcluster-src-n1" for 20 minutes

- Pausing x-cluster replication
  2023-10-25 20:37:58 Changing state of xcluster replication SRC-TO-TGT to Paused
  2023-10-25 20:38:10 Xcluster replication SRC-TO-TGT is now Paused
  2023-10-25 20:38:10 Changing state of xcluster replication TGT-TO-SRC to Paused
  2023-10-25 20:38:22 Xcluster replication TGT-TO-SRC is now Paused
  2 x-cluster streams are currently paused

2023-10-25 20:38:22 Shutting down DB server
2023-10-25 20:39:09 Server shut down and ready for maintenance

2023-10-25 20:39:09 Process completed successfully - exiting with code 0
```
