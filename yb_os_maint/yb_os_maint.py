# coding: utf8
# !/usr/bin/python
###############################################################
## Application Control for use with UNIX Currency Automation ##
###############################################################

Version = "2.04"

''' ---------------------- Change log ----------------------
V1.0 - Initial version :  08/09/2022 Original Author: Mike LaSpina - Yugabyte
V1.1 - Re-worked checking for YBA server based on the 1st services in the list
V1.2 - Added the following:
    file logging - pass '-t' to log output to terminal instead - usefule for dry runs
    Use first service in list to determine is we are on a YBA server (API is not availble if we are resuming YBA)
    Better error handling and output logging for starting/stopping YBA services check
    Skip YBA servers in list that we can't talk to (DEV/QA/PROD are firewalled from each other)
v1.3
    Added New parameter  '-l' or '--log_file_directory'.  Output will go to stdout if not specified
    Removed the '-t' parameter for terminal output
    Added tablet balance check and basic underreplicated checks
v1.4
    Added master and tserver lag checks via metrics - lag > xxx_LAG_THRESHOLD_SECONDS will prevent node from stopping
v1.5
    Changed tablet balance check logic to use tabet-servers screen from master in UI.
     Checking for text in TABLET_BALANCE_TEXT variable on web page
v1.6
    Fixed misplaced quote in master lag and added logging of promethues lag queries
v1.7
    Added encrypted host and key list file
    Added --health option which forces '--stop' and '--dryrun'
v1.8
    Converted subprocess.run to subprocess.check_output for 2.7 compatibility
v1.9
    Added logic to step down from HTTPS to HTTP for prometeus queries as some envs are configured this way
v1.10
    Changed return code to success when host is neither YBA nor DB node
v1.11
    Refactored to split out health check into it's own function
    Changed logic to use env variables YBA_HOST, API_TOKEN and CUST_UUID for YBA info
    Removed capability to look at multiple YBAs as values are now provided by env variables
v1.12
    Added error handling for missing environment variables
v1.13
    Changed logic to look for '.yba*.rc' in ENV_FILE_PATH and use those for host/token/cust
     script will error if no or more than 1 file is found.
v1.14
    Added check to wait_for_task to ensure task succeeded - otherwise bail with error
    Added sleep (currently 30 seconds) after task completes see 'TASK_COMPLETE_WAIT_TIME_SECONDS'
    Added action to log file name file name will now end with '_action.log' one of (health, stop, resume)
v1.15
    Reduced task completion sleep time to 10 seconds from 30
    Added random sleep (up to 30 seconds) prior to altering replication
    Added -f / --fix parameter to clear out placementModificationTaskUuid
    Added Warnings on healthcheck if isMaster or isTserver is false
    Added check to ensure the list of deadnodes returned are actually in the universe
       After a node is removed, it still shows as a dead node in the health-check endpoint
    Healthcheck now fails if placementModificationTaskUuid exists and is not blank
v1.16
    Added error checking/retries to alter_replication and refactored to simplify
v 1.17
    Added timestamps to various logging messages
    Added check on prometheus queries for results - brand new clusters will not have lag info
v 1.18
    Bypass master and tserver replication lag check if tablet count is zero
v 1.19
    Added maintenance widow  - create or extend for 20 mins on stop, remove on resume
v 1.20
    Decoupled healthcheck from having to run on node - now accepts a universe name or 'ALL'.  If running
      on a node, that node's universe is used if no other node (or ALL) is specified.
    Added --verify (-v) option to ensure master and tserver are in correct state per YBA
v 1.21
    Added YBA connectivity check and retry to start of script
    Refactored --verify function to make more readable
v 1.22
    Added check in clusters list for PlacementTaskUUID check.
    Added master stepdown call to yb-admin prior to node shutdown see 'LEADER_STEP_DOWN_COMMAND' variable
v 1.23
    Added variables for yb-admin command (YB_ADMIN_COMMAND) and tls_dir (YB_ADMIN_TLS_DIR)
    Modified logic to retry xcluster pause/resume when alter replication task fails
v 1.24
    Added code to deal with change in xcluster endpoint return in YBA 2.18.  Rather than status of
      Paused/Running, YBA 2.18 leaves status as 'Running' and introduced a 'paused' field in the return
v 1.25
    Fixes spelling typos, added doc
v 1.26
    New functionality to stop/resume all nodes in a region or region and availability zonw
    Pause/resume of xcluster can now be disabled via param
    Creation/deletion of maintenance window can now be disabled via param
    Added the following parameters:
      --region : Stops or Resumes all nodes in the given region - only applies to stop/resume.  Required if passing availbility_zone
      --availability_zone : Stops or Resumes all nodes in the given AZ - only applies to stop/resume.
      --skip_xcluster : Skips pausing and resuming of xCluster - only applies to stop/resume (forced when shutting down a region / az)
      --skip_maint_window: Skips maintenance window creation/removal - only applies to stop/resume (forced when shutting down a region / az)
v 1.27
    Added check for privateIP to match hostname in get_db_nodes_and_universe as it may contain a name in some cases
    Shored up yb-admin command to look up IP if master leader endpoint returns a hostname
     shutdown now continues if stepdown fails or errors out
v 1.28 (Last fixes from Mike L)
    Added generic retry logic, log timestamp option. Wait for tasks that may be "Running" at 100%.
v 1.29
    Allow node ops (as no-error,no-op) from un-configured nodes. 
v 1.30
    BugFix - for universe==None case for functionality for "New" nodes
v 1.31, 1.32 , 1.33, 1.34
    BugFix - check if YBA before giving up on "New node"
v 1.35 -> 2.01 
    Major Re-factor to O-O. YBA, YBDB-node, multi-node; Add check under-replicated tablets.
v 2.03, 2.04
    Enable --region, if DB node errors out, assume "unconfigured" node
'''

import argparse
import requests
import json
import socket
import time
import subprocess
import traceback
from datetime import datetime, timedelta
import urllib3
import requests.packages
import sys
import os
import fnmatch
import random
import copy
from urllib3.exceptions import InsecureRequestWarning

## Return value constants
OTHER_ERROR = 1
OTHER_SUCCESS = 0
NODE_DB_SUCCESS = 0
NODE_DB_ERROR = 1
NODE_YBA_SUCCESS = 0
NODE_YBA_ERROR = 1

## Globals
MIN_MOST_RECENT_UPTIME_SECONDS = 60
TASK_WAIT_TIME_SECONDS = 2.0
TASK_COMPLETE_WAIT_TIME_SECONDS = 10.0

TABLET_BALANCE_TEXT = 'Cluster Load is Balanced'
PROMETHEUS_PORT = 9090
MASTER_LAG_THRESHOLD_SECONDS = 60
TSERVER_LAG_THRESHOLD_SECONDS = 60
LAG_LOOKBACK_MINUTES = 10
ENV_FILE_PATH = '/home/yugabyte/'
ENV_FILE_PATTERN = '.yba*.rc'
PLACEMENT_TASK_FIELD = 'placementModificationTaskUuid'
MAX_TIME_TO_SLEEP_SECONDS = 30
LOG_TIME_FORMAT = "%Y-%m-%d %H:%M:%S"
MAINTENANCE_WINDOW_NAME = 'OS Patching - '
MAINTENANCE_WINDOW_DURATION_MINUTES = 20
YB_ADMIN_COMMAND = '/home/yugabyte/tserver/bin/yb-admin'
YB_ADMIN_TLS_DIR = '/home/yugabyte/yugabyte-tls-config'
LEADER_STEP_DOWN_COMMAND = '{} -master_addresses {{}} -certs_dir_name {}'.format(YB_ADMIN_COMMAND, YB_ADMIN_TLS_DIR)
LOCALHOST = '<localhost>'

# Global scope variables - do not change!
LOG_TO_TERMINAL = True
LOG_FILE = None
## Custom Functions

# Log messages to output - all messages go thru this function
def log(msg, isError=False, logTime=False):
    output_msg = ''
    if isError:
        output_msg = '*** Error ***: '
    if logTime:
        output_msg = output_msg + ' ' + datetime.now().strftime(LOG_TIME_FORMAT) + " "
    output_msg = output_msg + str(msg) # Stringify in case msg was of type "exception"

    if LOG_TO_TERMINAL:
        print(output_msg)
    else:
        LOG_FILE.write(output_msg + '\n')

def retry_successful(retriable_function_call, params=None, retry:int=10, verbose=False, sleep:float=.5, fatal=False):
    for i in range(retry):
        try:
            verbose and log("Attempt {} running {}. Will wait {} sec.".format(i+1, retriable_function_call.__name__,sleep * i),logTime=True)
            time.sleep(sleep * i)
            retval = retriable_function_call(*params)
            verbose and retval != None and log(
                "  Returned {} from called function {} on attempt {}".format(retval, retriable_function_call.__name__, i+1))
            return True
        except Exception as errorMsg:
            preserve_errmsg = errorMsg
            #tb_info = traceback.extract_tb(errorMsg.__traceback__)
            ## iterate over the traceback entries
            #for tb in tb_info:
            #    file_name, line_no, func_name, code = tb
            #    print(f"Trace:Error occurred in {file_name} at line {line_no}")
            verbose and log("  Hit exception {} in attempt {}".format(errorMsg, i+1))
            if fatal and i == (retry - 1):
                raise  # Re-raise current exception
            continue
    return False

def get_node_ip(addr): # Returns dotted-decimal addr
    try:
        socket.inet_aton(addr)
        return(addr)
    except:
        return(socket.gethostbyname(addr))

#-------------- Class definitions ---------------------------------------
class NotMyTypeException(Exception):
    pass

class Universe_class:
    def __init__(self,YBA_API,json):
        self.YBA_API        = YBA_API
        self.json           = json
        self.args           = YBA_API.args
        self.nodeDetailsSet = json['universeDetails']['nodeDetailsSet']
        self.UUID           = json["universeUUID"]
        self.name           = json["name"]
        self.universeDetails         = json['universeDetails']
        self.sourceXClusterConfigs   = json['universeDetails'].get('sourceXClusterConfigs')
        self.targetXClusterConfigs   = json['universeDetails'].get('targetXClusterConfigs')
        self.PLACEMENT_TASK_FIELD    = json['universeDetails'].get(PLACEMENT_TASK_FIELD)
        self.SKIP_DEAD_NODE_CHECK    = False


    def get_node_json(self,hostname,ip):
        for candidate_node in self.nodeDetailsSet:
            if str(candidate_node['nodeName']).upper() in hostname.upper() or hostname.upper() in \
                str(candidate_node['nodeName']).upper() or \
                    candidate_node['cloudInfo']['private_ip'] == ip or \
                    candidate_node['cloudInfo']['public_ip'] == ip or \
                    candidate_node['cloudInfo']['private_ip'].upper() in hostname.upper():
                return candidate_node
        return None
    
    def find_nodes_by_region_az(self,region:str,az:str):
        nodes  = []
        region = region.upper()
        if az is not None:
            az     = az.upper()
        for candidate_node in self.nodeDetailsSet:
            if candidate_node['cloudInfo']['region'].upper() == region:
                if az is None:
                    nodes.append(candidate_node)
                elif candidate_node['cloudInfo']['az'].upper() == az:
                        nodes.append(candidate_node)
        return nodes

    def get_master_leader_ip(self,include_port=False):
        j = self.YBA_API.get_universe_info(self.UUID,'/leader')
        if not isinstance(j, dict):
            raise Exception("Call to get leader IP returned {} instead of dict".format(type(j)))
        if not 'privateIP' in j:
            raise Exception("Could not determine master leader - privateIP was not found in {}".format(j))
        if include_port:
            return(get_node_ip(j['privateIP']) + ":" + str(self.nodeDetailsSet[0]["masterHttpPort"]))
        return(get_node_ip(j['privateIP']))
    
    def check_under_replicated_tablets(self):
        """
        http://172.31.23.16:7000/api/v1/tablet-under-replication
        Sample output
        {"underreplicated_tablets":[{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"c0acc61a6f874a489d113494ab266c39","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"4bcb4184a80c4c0dbdd5bd07063fe66b","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"98c7852212ad4919b726ad5ad8f27025","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"70d12ac57b9449b2b04f30d4a0a5dbc7","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"dc97c6805d234c88a39dab8443eb2cda","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"b6bb01582efe47008a583fcaa4698258","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"7dcdeefeaa48457ea7155f572cc9aaee","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"c9eaf49983784050b41aa981706b9648","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"fea04949daee498fbddb6ae5f5a54d2e","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"86996c4c524e48c397a5733bfd599cad","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"6cb2f58212ae4495b96f30979839ec31","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"7dff77b01e8c4c528b4047af0d64913c","tablet_uuid":"b9cae88995fc4898910c838c0a5c302c","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"000033e9000030008000000000004000","tablet_uuid":"adb412ac66f346db9176abbd4ec3583b","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"000033e9000030008000000000004000","tablet_uuid":"ffc63a74eb94448fb76edb2cd3d1f154","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"000033e9000030008000000000004000","tablet_uuid":"d8e632f4748343e4be6534ffdddfef9e","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]},{"table_uuid":"000033e9000030008000000000004000","tablet_uuid":"c6069b51565e4aab9f3f049d4d876684","underreplicated_placements":["0bc2fe62-3180-48b9-99de-ebd84ae0af8c"]}]}

        Check if the system is under replicated. It will go after master(leader) and curl /api/v1/tablet-under-replication.
        It will print out a list of under replicated tablet - table.

        :return 0 if it has no under replicated
                >0 return number of tablet is under_replicated
                -1 if it can't get the number of under replicated tablet
        """

        master_node = self.get_master_leader_ip(include_port=True)
        try:  # try both http and https endpoints
            resp = requests.get('https://' + master_node + '/api/v1/tablet-under-replication',
                                verify=False)
            under_replicated_tablets = resp.json()
        except:
            resp = requests.get('http://' + master_node + '/api/v1/tablet-under-replication')
            under_replicated_tablets = resp.json()

        under_replicated_tablets_list = under_replicated_tablets['underreplicated_tablets']

        # Doing it if there is more than 0's under replicated tablets
        if len(under_replicated_tablets_list) > 0:
            log(f"\t\t tablet_uuid \t\t - \t\t table_uuid")
            for under in under_replicated_tablets_list[0:5]:
                log(f"\t{under['tablet_uuid']} \t - \t {under['table_uuid']}")
            if len(under_replicated_tablets_list)  > 5:
                log("\t......truncated to 5 tablets .......")

            log("\n")
            log(f"Total # of under replicated tablets: {len(under_replicated_tablets_list)}")
            raise Exception(len(under_replicated_tablets_list) + " Under-replicated tablets")

        return len(under_replicated_tablets_list)

    def get_dead_node_count(self):
        if self.SKIP_DEAD_NODE_CHECK:
            return 0
        
        nlist = self.YBA_API.get_universe_info(self.UUID,'/status')
        dead_nodes    = 0
        dead_masters  = 0
        dead_tservers = 0
        # this one comes back as a dict instead of proper JSON, also last key is uuid, which we need to ignore
        for n in nlist:
            if n != 'universe_uuid':
                if nlist[n]['node_status'] != 'Live':
                    log('  Node ' + n + ' is not alive', True)
                    dead_nodes += 1

                for nodes in self.nodeDetailsSet:
                    if nodes['nodeName'] == n:
                        if nodes['isMaster'] and not nlist[n]['master_alive']:
                            log('  Node ' + n + ' master is not alive', True)
                            dead_masters += 1

                        if nodes['isTserver'] and not nlist[n]['tserver_alive']:
                            log('  Node ' + n + ' tserver is not alive', True)
                            dead_tservers += 1
        return max(dead_nodes, dead_masters, dead_tservers)

    def health_check(self):
        log('Performing health checks for universe {}, UUID {}...'.format(self.name, self.UUID))
        log('- Checking for task placement UUID',logTime=True)
        errcount = 0;
        if (PLACEMENT_TASK_FIELD in self.json and len(self.PLACEMENT_TASK_FIELD) > 0) or \
                (PLACEMENT_TASK_FIELD in self.universeDetails and len(self.PLACEMENT_TASK_FIELD) > 0):
            log('Non-empty ' + PLACEMENT_TASK_FIELD + "=" + self.PLACEMENT_TASK_FIELD + ' found in universe', True)
            errcount +=1
        else:
            log('  Passed placement task test')

        log('- Checking for dead nodes, tservers and masters')
        errcount += self.get_dead_node_count()

        if errcount == 0:
            log('  All nodes, masters and t-servers are alive.')

        log('- Checking for master and tablet health')

        master_node = None
        for masternode in self.nodeDetailsSet:
            if masternode['isMaster'] and masternode['state'] == 'Live':
                master_node = masternode['cloudInfo']['private_ip'] + ':' + str(masternode['masterHttpPort'])
                break
        
        if master_node is None:
            log("No master nodes found - FAILED health check.",isError=True)
            raise Exception("Health check failed")

        try:  # try both http and https endpoints
            resp = requests.get('https://' + master_node + '/api/v1/health-check',verify=False)
            hc = resp.json()
        except:
            resp = requests.get('http://' + master_node + '/api/v1/health-check')
            hc = resp.json()
        hc_errs = 0
        for n in hc:
            if n == 'most_recent_uptime':
                log('  Most recent uptime: ' + str(hc[n]) + ' seconds')
                if hc[n] < MIN_MOST_RECENT_UPTIME_SECONDS:
                    log('All nodes in the cluster have not been up for the minumim of ' + str(
                        MIN_MOST_RECENT_UPTIME_SECONDS) + ' seconds', True)
                    errcount += 1
                    hc_errs += 1
            elif n == 'dead_nodes':
                if len(hc[n]) > 0:
                    isInUniverse = False
                    numDeadNodes = 0
                    for uid in hc[n]:
                        for tnode in self.nodeDetailsSet:
                            if 'nodeUuid' in tnode and uid == tnode['nodeUuid'].replace('-', ''):
                                log('Health check found the following dead node in the universe: ' + uid, True)
                                isInUniverse = True
                                numDeadNodes += 1
                                errcount += 1
                                hc_errs += 1
                                break
                    if not isInUniverse and numDeadNodes == 0:
                        log('  Found the following dead nodes that are not currently in the universe - continuing:')
                        log('   ' + json.dumps(hc[n]))
            elif len(hc[n]) > 0:
                log('Health check found an issue with ' + n + ' - see below', True)
                log(json.dumps(hc[n], indent=2))
                errcount += 1
                hc_errs += 1
        if hc_errs == 0:
            log('  No under replicated or leaderless tablets found')

        ## Check tablet balance
        log('- Checking tablet health')
        log('  --- Tablet Report --')
        tabs = None
        totalTablets = 0
        try:  # try both http and https endpoints
            resp = requests.get('https://' + master_node + '/api/v1/tablet-servers',verify=False)
            tabs = resp.json()
        except:
            resp = requests.get('http://' + master_node + '/api/v1/tablet-servers')
            tabs = resp.json()

        for uid in tabs:
            for svr in tabs[uid]:
                if tabs[uid][svr]['status'] != 'ALIVE':
                    log('  TServer ' + svr + ' is not alive and likely a deprecated node - skipping')
                else:
                    log('  TServer {} Active tablets: {}, User tablets: {}, User tablet leaders: {}, System Tablets: {}, System tablet leaders: {}'.format(
                        svr,
                        tabs[uid][svr]['active_tablets'],
                        tabs[uid][svr]['user_tablets_total'],
                        tabs[uid][svr]['user_tablets_leaders'],
                        tabs[uid][svr]['system_tablets_total'],
                        tabs[uid][svr]['system_tablets_leaders']
                        ))
                    totalTablets = totalTablets + tabs[uid][svr]['active_tablets'] + tabs[uid][svr]['user_tablets_total'] + tabs[uid][svr]['user_tablets_leaders']

        htmlresp = None
        try:  # try both http and https endpoints
            htmlresp = requests.get('https://' + master_node + '/tablet-servers',verify=False)
        except:
            htmlresp = requests.get('http://' + master_node + '/tablet-servers')

        if(TABLET_BALANCE_TEXT) in htmlresp.text:
            log('  Passed tablet balance check - YBA is reporting the following: ' + TABLET_BALANCE_TEXT)
        else:
            errcount += 1
            log('Tablet Balance check failed', True)

        # Check tablet lag
        if totalTablets > 0:
            log('  Checking replication lag for t-servers')
            promnodes =''
            for tnode in self.nodeDetailsSet:
                if tnode['isTserver']:
                    promnodes += tnode['nodeName'] + '|'
            promql = 'max(max_over_time(follower_lag_ms{exported_instance=~"' + promnodes +\
                        '",export_type="tserver_export"}[' + str(LAG_LOOKBACK_MINUTES) + 'm]))'
            log('  Executing the following Prometheus query:')
            log('   ' + promql)
            resp = requests.get(self.YBA_API.promhost, params={'query':promql}, verify=False)
            metrics = resp.json()
            lag = float(0.00)
            if 'data' in metrics and \
                'result' in metrics['data'] and \
                len(metrics['data']['result']) > 0 and \
                'value' in metrics['data']['result'][0] and \
                len(metrics['data']['result'][0]['value']) > 1:
                lag = float(metrics['data']['result'][0]['value'][1]) / 1000
            if lag > TSERVER_LAG_THRESHOLD_SECONDS:
                log('Check failed - t-server lag of {} seconds is above threshhold of {}'.format(lag, TSERVER_LAG_THRESHOLD_SECONDS), True)
                errcount+=1
            else:
                log('  Check passed - t-server lag of {} seconds is below threshhold of {}'.format(lag, TSERVER_LAG_THRESHOLD_SECONDS))
        else:
            log('  Tablet count in universe is zero - bypassing t-server replication lag check')

        log('- Checking  master health')
        # Check underreplicated masters (master list should equal RF for universe and lag should be below threshold
        log('  Checking for underreplicated masters')
        master_list = self.YBA_API.get_universe_info(self.UUID,'/masters')
        num_masters = len(str(master_list).split(','))
        if self.universeDetails['clusters'][0]['userIntent']['replicationFactor'] == num_masters:
            log('  Check passed - cluster has RF of {} and {} masters'.format(
                self.universeDetails['clusters'][0]['userIntent']['replicationFactor'],
                num_masters))
        else:
            log('Check failed - cluster has RF of {} and {} masters'.format(
                self.universeDetails['clusters'][0]['userIntent']['replicationFactor'],
                num_masters), True)
            errcount+=1

        # Check master lag
        if totalTablets > 0:
            log('  Checking replication lag for masters')
            promnodes =''
            for mnode in self.nodeDetailsSet:
                if mnode['isMaster']:
                    promnodes += mnode['nodeName'] + '|'
            promql = 'max(max_over_time(follower_lag_ms{exported_instance=~"' + promnodes +\
                        '",export_type="master_export"}[' + str(LAG_LOOKBACK_MINUTES) + 'm]))'
            log('  Executing the following Prometheus query:')
            log('   ' + promql)
            resp = requests.get(self.YBA_API.promhost, params={'query':promql}, verify=False)
            metrics = resp.json()
            lag = float(0.00)
            if 'data' in metrics and \
                'result' in metrics['data'] and \
                len(metrics['data']['result']) > 0 and \
                'value' in metrics['data']['result'][0] and \
                len(metrics['data']['result'][0]['value']) > 1:
                lag = float(metrics['data']['result'][0]['value'][1]) / 1000
            if lag > MASTER_LAG_THRESHOLD_SECONDS:
                log('Check failed - master lag of {} seconds is above threshhold of {}'.format(lag, MASTER_LAG_THRESHOLD_SECONDS), True)
                errcount+=1
            else:
                log('  Check passed - master lag of {} seconds is below threshhold of {}'.format(lag, MASTER_LAG_THRESHOLD_SECONDS))
        else:
            log('  Tablet count in universe is zero - bypassing master replication lag check')

        ## End health checks,
        if errcount > 0:
            log('--- Health check has failed - ' + str(
                errcount) + ' errors were detected.',isError=True,logTime=True)
            raise Exception("Health check failed")
        else:
            log('--- Health check for universe "{}" completed with no issues'.format(self.name))
            return

    def fix(self):
        log('Fixing the following items in the universe: ' + str(self.args.fix))
        mods = []
        f = self.universeDetails
        if 'placement' in self.args.fix:
            if PLACEMENT_TASK_FIELD in f and len(f[PLACEMENT_TASK_FIELD]) > 0:
                f[PLACEMENT_TASK_FIELD] = ''
                mods.append('placement')
                # This requires running PSQL with:
                #    "update universe set universe_details_json='{}' where universe.universe_uuid='{}';"
                log("Sorry - FIX for placement is not implemented.", isError=True)
                raise Exception("Not Implemented")

        if len(mods) == 0:
            log('No items exist to fix')
            return
        
        log('Updating universe config with the following fixed items: ' + str(mods))
        if 'tserverGFlags' not in f:
            f['tserverGFlags'] = {"vmodule": "secure1*"}
        if 'masterGFlags' not in f:
            f['masterGFlags'] = {"vmodule": "secure1*"}
        f['upgradeOption'] = "Non-Restart"
        f['sleepAfterMasterRestartMillis'] = 0
        f['sleepAfterTServerRestartMillis'] = 0
        #f['kubernetesUpgradeSupported'] = False
        response = requests.post(self.YBA_API.api_host + '/api/v1/customers/' + self.YBA_API.customer_uuid + '/universes/' +
                                self.UUID + '/upgrade/gflags',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.YBA_API.api_token},
            json=f,verify=False)
        task = response.json()
        if  task.get('taskUUID') is None:
            log("gflag task error:{}".format(task))
            raise Exception("Failed to create gflag update task")
        if retry_successful(self.YBA_API.wait_for_task, params=[ task['taskUUID'] ],sleep=TASK_COMPLETE_WAIT_TIME_SECONDS,verbose=True,retry=15):
            log(' Server items fixed', logTime=True)
            restarted = True
        else:
            log("Fix task failed",isError=True,logTime=True)
            raise Exception("Fix task failed")

    def Pause_xCluster_Replication(self):
        ## pause source replication
        log('\n- Pausing x-cluster replication', logTime=True)
        paused_count = 0
        ## pause source replication
        if 'sourceXClusterConfigs' in self.universeDetails:
            for rpl in self.sourceXClusterConfigs:
                ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
                time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
                if self.YBA_API.alter_replication('pause', rpl):
                    paused_count += 1
                else:
                    raise Exception("Failed to pause source x-cluster replication")
        ## pause target replication
        if 'targetXClusterConfigs' in self.universeDetails:
            for rpl in self.targetXClusterConfigs:
                ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
                time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
                if self.YBA_API.alter_replication('pause', rpl):
                    paused_count += 1
                else:
                    raise Exception("Failed to pause target x-cluster replication")
        if paused_count > 0:
            log('  ' + str(paused_count) + ' x-cluster streams are currently paused', logTime=True)
        else:
            log('  No x-cluster replications were found to pause', logTime=True)

    def Resume_xCluster_replication(self):
        ## resume source replication
        log('\n- Resuming x-cluster replication')
        resume_count = 0
        if 'sourceXClusterConfigs' in self.universeDetails:
            for rpl in self.sourceXClusterConfigs:
                if self.args.dryrun:
                    response = requests.get(
                        self.YBA_API.api_host + '/api/customers/' + self.YBA_API.customer_uuid + '/xcluster_configs/' + rpl,
                        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.YBA_API.api_token},verify=False)
                    xcl_cfg = response.json()
                    xcCurrState = ''
                    if 'paused' in xcl_cfg:
                        if xcl_cfg['paused']:
                            xcCurrState = 'Paused'
                        else:
                            xcCurrState = 'Running'
                    else:
                        xcCurrState = xcl_cfg['status']
                    log('  Replication ' + xcl_cfg['name'] + ' is in state ' + xcCurrState)
                else:
                    # Pause/resume as directed and if not in the correct state
                    if self.YBA_API.alter_replication('resume', rpl):
                        resume_count += 1
                    else:
                        raise Exception("Failed to resume x-cluster replication")

        ## resume target replication
        if 'targetXClusterConfigs' in self.universeDetails:
            for rpl in self.targetXClusterConfigs:
                if self.args.dryrun:
                    response = requests.get(
                        self.YBA_API.api_host + '/api/customers/' + self.YBA_API.customer_uuid + '/xcluster_configs/' + rpl,
                        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.YBA_API.api_token},verify=False)
                    xcl_cfg = response.json()
                    xcCurrState = ''
                    if 'paused' in xcl_cfg:
                        if xcl_cfg['paused']:
                            xcCurrState = 'Paused'
                        else:
                            xcCurrState = 'Running'
                    else:
                        xcCurrState = xcl_cfg['status']
                    log('  Replication ' + xcl_cfg['name'] + ' is in state ' + xcCurrState)
                else:
                    # Pause/resume as directed and if not in the correct state
                    if self.YBA_API.alter_replication('resume', rpl):
                        resume_count += 1
                    else:
                        raise Exception("Failed to resume x-cluster replication")
        if resume_count > 0:
            if self.args.dryrun:
                log('  ' + str(resume_count) + ' x-cluster streams were found, but not resumed due to dry run')
            else:
                log('  ' + str(resume_count) + ' x-cluster streams are now running')
        else:
                log('  No x-cluster replications were found to resume')

#-------------------------------------------------------------------------------------------
class Multiple_Nodes_Class:
    # Used when --universe or (--region + --az) is specified
    # In these cases, ignore WHAt node we are running on, and operate on the requested nodes.
    def __init__(self,host,YBA_API,args):
        self.YBA_API  = YBA_API
        self.args     = args
        self.nodeList = []
        self.universe = None
        
        if args.health and args.health.upper() == "ALL" and not args.universe:
            self.args.universe = "ALL"
        if self.args.universe:
            self.YBA_API.Initialize()
            if args.universe.upper() == "ALL"  and args.health:
                self.universe = args.universe.upper() 
                pass
            else:
                self.universe = self.YBA_API.find_universe_by_name_or_uuid(self.args.universe)
                if self.universe is None:
                    log("Could not find a universe named "+ args.universe,isError=True)
                    raise Exception("Specified universe could not be found")

        if self.args.region :
            self.region = args.region
            self.az     = args.availability_zone
            if self.az is not None  and "=" in self.az:
                log("WARNING: specified AZ '" + self.az + "' contains an '=' sign .. does not look right. check your '-a' or '--availability_zone' param")
            self.YBA_API.Initialize()
            if self.universe is None:
                self.universe = self.YBA_API.find_universe_by_region_az(args.region, args.availability_zone)
                if self.universe is None:
                    log("Could not determine universe based on Region+AZ. Please specify --universe also",isError=True)
                    raise Exception("Arguments incorrect/insufficient")
            for n in  self.universe.find_nodes_by_region_az(args.region, args.availability_zone):
                self.nodeList.append(YB_Data_Node.construct_from_json(n,self.universe,YBA_API,args))
            if len(self.nodeList) == 0:
               raise Exception("Did not find any nodes to operate on in the specified region/az")
        if self.universe is None:
            raise NotMyTypeException("Neither --universe nor (--region + --az) were properly specified")
    
    def health(self):
        if isinstance(self.universe,str) and self.universe.upper() == "ALL":
            fail_count = 0
            for u in self.YBA_API.universe_list:
                try:
                    u.health_check()
                except:
                    log("Universe "+ u.name + "(" + u.UUID +") failed health check")
                    fail_count += 1
                log("----------------------------") # Separator between health checks...
            if fail_count == 0:
                return
            raise Exception(str(fail_count) + " universes failed health check")
        
        self.universe.health_check()

    def fix(self):
        self.universe.fix()

    def stop(self):
        if len(self.nodeList) == 0:
            raise Exception("Did not find any nodes to operate on")
        log("performing STOP on "+str(len(self.nodeList)) + ' nodes.')
        if self.args.dryrun:
            log("Dry Run : Not performing STOP")
            return
        self.universe.SKIP_DEAD_NODE_CHECK = True

        try:
            if self.args.skip_xcluster:
                pass
            else:
                self.universe.Pause_xCluster_Replication()

            for n in self.nodeList: # Stop tservers first
                n.args.skip_xcluster = True # don't let individual node do it.
                if n.isMaster:
                    continue # do it in the next loop
                if  n.node_json['state'] != "Live":
                    log(n.hostname + " is in " +  n.node_json['state'] + " state. Skipping", logTime=True)
                    continue
                n.stop()
                retry_successful(self.universe.check_under_replicated_tablets,params=[],fatal=True,sleep=30,verbose=True)
            for n in self.nodeList: # Stop masters 
                if n.isMaster:
                    if  n.node_json['state'] != "Live":
                        log(n.hostname + " is in " +  n.node_json['state'] + " state. Skipping", logTime=True)
                        continue
                    n.stop()
                    retry_successful(self.universe.check_under_replicated_tablets,params=[],fatal=True,sleep=30,verbose=True)

        except Exception as e:
            log("Stop multiple node encountered " + str(e),isError=True)
            raise

    def resume(self):
        if len(self.nodeList) == 0:
            raise Exception("Did not find any nodes to operate on")
        log("performing RESUME on "+str(len(self.nodeList)) + ' nodes.')
        if self.args.dryrun:
            log("Dry Run : Not performing RESUME")
            return
        for n in self.nodeList:
            if  n.node_json['state'] == "Live":
                    log(n.hostname + " is in " +  n.node_json['state'] + " state. Skipping", logTime=True)
                    continue
            n.args.skip_xcluster = True
            n.resume()
        if self.args.skip_xcluster:
            pass
        else:
            self.universe.Resume_xCluster_replication()

#-------------------------------------------------------------------------------------------
class YBA_Node:
    _YBA_PROCESS_STOP_LIST = ['yb-platform', 'prometheus', 'rh-nginx118-nginx', 'rh-postgresql10-postgresql']

    def __init__(self,host,YBA_API,args):
        #log('Checking if host ' + host + ' is YBA Instance')
        self.hostname = host
        self.YBA_API  = YBA_API
        self.args     = args
        try:
            r = subprocess.check_output("systemctl list-units --all '{}.service'".format(YBA_Node._YBA_PROCESS_STOP_LIST[0]), shell=True, stderr=subprocess.STDOUT)
            if '{}.service'.format(YBA_Node._YBA_PROCESS_STOP_LIST[0]) in str(r):
                self.setVersion()
                return None # Successful
            else:
                try:
                   r = subprocess.check_output("docker ps -f name=yugaware", shell=True, stderr=subprocess.STDOUT)
                   if "yugaware" in str(r):
                       log("This old-style docker-based YBA is NOT SUPPORTED",isError=True,logTime=True)
                       os._exit(9)
                except:
                    pass   
                raise NotMyTypeException(host + " is not a YBA node")
        except subprocess.CalledProcessError as e:
            log('Error checking for YBA process: ', isError=True,logTime=True)
            log(e.output,isError=True)
            raise NotMyTypeException(host + " is not a YBA node(Error getting services)")

    def setVersion(self):
        activePath = subprocess.check_output(['readlink','-f','/opt/yugabyte/software/active'],text=True)
        self.ybaVersion = activePath.rstrip('\n').split("/")[-1] # Last dir is a version like '2.18.5.2-b1'

    def health(self):
        if self.ybaVersion < '2.18.0':
            raise ValueError("'health' is not Implemented for YBA version " + self.ybaVersion + " (< 2.18)")
        try:
            status=subprocess.check_output(['yba-ctl','status'],stderr=subprocess.STDOUT,text=True) 
        except subprocess.CalledProcessError as e:
            log('Error checking for YBA status: ', isError=True,logTime=True)
            log(e.output,isError=True)
            raise # re-raise for caller's benefit 
        log(status,logTime=True)

    def resume(self):
        log(' Host is YBA Server - Starting up services...', logTime=True)
        if self.ybaVersion >= '2.18.0':
            try:
                status=subprocess.check_output(['yba-ctl','start'],  stderr=subprocess.STDOUT) # No output
                time.sleep(2)
                self.health()
                return(True)
            except subprocess.CalledProcessError as e:
                log('  yba-ctl start failed. Err:{}'.format(str(e)),logTime=True,isError=True)
                exit(NODE_YBA_ERROR)
        # Older YBA - shut down individual services 
        for svc in reversed(YBA_Node.YBA_PROCESS_STOP_LIST):
            try:
                # This call triggers an error if the process is not active.
                log('  Stopping service ' + svc)
                status = subprocess.check_output('systemctl is-active {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                log('  Service {} is already running - skipping'.format(svc))
            except subprocess.CalledProcessError as e: # This is the code path if the service is not running
                try:
                    o = subprocess.check_output('systemctl start {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                    log('  Service ' + svc + ' started', logTime=True)
                except subprocess.CalledProcessError as err:
                    log('Error starting service' + svc + '- see output below', isError=True)
                    log(err)
                    log(' Process failed - exiting with code ' + str(NODE_YBA_ERROR), logTime=True)
                    exit(NODE_YBA_ERROR)

    def stop(self):
        log(' Host is YBA Server {} - Shutting down services...'.format(self.ybaVersion), logTime=True)
        if self.ybaVersion >= '2.18.0':
            try:
                status=subprocess.check_output(['yba-ctl','stop'],  stderr=subprocess.STDOUT) # No output
                time.sleep(2)
                self.health()
                return(True)
            except subprocess.CalledProcessError as e:
                log('  yba-ctl stop failed - skipping. Err:{}'.format(str(e)),logTime=True)
                exit(NODE_YBA_ERROR)

        for svc in YBA_Node.YBA_PROCESS_STOP_LIST:
            try:
                # This call triggers an error if the process is not active.
                log('  Stopping service ' + svc)
                status = subprocess.check_output('systemctl is-active {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                try:
                    o = subprocess.check_output('systemctl stop {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                    log('  Service ' + svc + ' stopped', logTime=True)
                except subprocess.CalledProcessError as err:
                    log('Error stopping service' + svc + '- see output below', isError=True)
                    log(err)
                    log('\nProcess failed - exiting with code ' + str(NODE_YBA_ERROR))
                    exit(NODE_YBA_ERROR)
            except subprocess.CalledProcessError as e:
                log('  Service {} is not running - skipping'.format(svc))
#-------------------------------------------------------------------------------------------
class YBA_API_CLASS:
    def __init__(self,api_host, customer_uuid, api_token,args):
        self.api_host      = api_host
        self.customer_uuid = customer_uuid
        self.api_token     = api_token
        self.args          = args
        self.universe_list = []
        self.promhost      = None
        self.initialized   = False

    def Initialize(self):
        if self.initialized:
            return
        retry_successful(self.get_customers, params=[], retry=5,verbose=True,sleep=5,fatal=True)
        log('Retrieving Universes from YBA server at ' +self.api_host)
        # Get all universes on YBA deployment.  Note that a list of nodes is included here, so we return the entire universes array
        response = requests.get(self.api_host + '/api/customers/' + self.customer_uuid + '/universes',
                                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False)
        for univ in response.json():
            self.universe_list.append(Universe_class(self,univ)) # List of "Universe_class" objects 

        # put together Prometheus URL by stripping off existing port of API server if specified and appending proper port
        # Then try https, if fails, step down to http
        tmp_url = self.api_host.split(':')
        promhost = tmp_url[0] + ':' + tmp_url[1] + ':' + str(PROMETHEUS_PORT) + '/api/v1/query'
        try:
            log('\nChecking for prometheus host at {}'.format(promhost))
            resp = requests.get(promhost, params={'query': 'min(node_boot_time_seconds)'}, verify=False)
        except:
            if 'https' in promhost:
                promhost = promhost.replace('https', 'http')
                log('Could not contact prometheus host using HTTPS.  Trying insecure connection at {}'.format(promhost))
                resp = requests.get(promhost, params={'query': 'min(node_boot_time_seconds)'}, verify=False)
            else:
                log('Could not contact prometheus host at {}'.format(promhost), isError=True)
                errcount += 1;
        log('Using prometheus host at {}\n'.format(promhost))
        self.promhost=promhost
        self.initialized = True

    def find_universe_by_name_or_uuid(self,lookfor_name:str=None):
        if lookfor_name is None:
            return None
        lookfor_name = lookfor_name.upper()
        for candidate_universe in self.universe_list:
            if lookfor_name == candidate_universe.name.upper() \
                  or lookfor_name == candidate_universe.UUID.upper():
                return candidate_universe
        return None

    def find_universe_by_region_az(self,region,az):
        univ_dict   = {} # Contains tuples {univ-UUID: Number-found}
        region      = region.upper()
        if az is not None:
            az          = az.upper()
        for candidate_universe in self.universe_list:
            #  u[1]['universeDetails']['clusters'][0]['placementInfo']['cloudList'][0]['regionList'][0]['name']
            for cluster in candidate_universe.universeDetails['clusters']:
                for cloud in cluster['placementInfo']['cloudList']:
                    for reg in cloud['regionList']:
                        #print (candidate_universe['name']," : ",reg['name'])
                        if region in reg['name'].upper()  or  region in reg['code'].upper():
                            if az is None:
                                univ_dict[candidate_universe.UUID] = univ_dict.get(candidate_universe.UUID , 0) + 1
                                break
                            else:
                                for candidate_az in reg['azList']:
                                    #print ("       az ",candidate_az['name'])
                                    if az in candidate_az['name'].upper():
                                        univ_dict[candidate_universe.UUID] = univ_dict.get(candidate_universe.UUID , 0) + 1
                                        break

        if len(univ_dict) == 1:
            return self.find_universe_by_name_or_uuid(next(iter(univ_dict))) # univ object corresponding to First key in dict
        log("Found "+ str(len(univ_dict)) + " Universes for region/az:"+region + "/"+ az, isError=True)
        return None


    def find_universe_for_node(self,hostname=None,ip=None):
        for universe in self.universe_list:
            node_json = universe.get_node_json(hostname,ip)
            if node_json is None:
                continue # to the next universe 
            return universe,node_json
        return None,None

  
    def get_universe_info(self,univ_uuid,endpoint):
        resp = requests.get(
                self.api_host + '/api/v1/customers/' + self.customer_uuid + '/universes/' + univ_uuid + endpoint,
                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False)
        return resp.json()
    
    def maintenance_window(self, node, action):
        host = node.hostname
        w_id = self.find_window_by_name(host)
        if action == 'create':
            mins_to_add = timedelta(minutes=MAINTENANCE_WINDOW_DURATION_MINUTES)
            j = {"customerUUID" : self.customer_uuid,
                "name" : MAINTENANCE_WINDOW_NAME + host,
                "description" : MAINTENANCE_WINDOW_NAME + host,
                "startTime": datetime.now().isoformat('T','seconds') + "Z",                #yyyy-MM-dd'T'HH:mm:ss'Z'
                "endTime": (datetime.now() + mins_to_add).isoformat('T','seconds') + "Z",
                "alertConfigurationFilter": {
                    "targetType": "UNIVERSE",
                    "target": {
                        "all": False,
                        "uuids": [node.universe.UUID]
                    }
                }
            }
            if w_id is not None:
                log('\n- Updating existing Maintenance window "{}" for {} minutes' \
                    .format(MAINTENANCE_WINDOW_NAME + host, str(MAINTENANCE_WINDOW_DURATION_MINUTES)) \
                    , logTime=True)
                j['uuid'] = w_id
                response = requests.put(
                    self.api_host + '/api/v1/customers/' + self.customer_uuid + '/maintenance_windows/' + w_id,
                    headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False,
                    json = j)
            else:
                log('\n- Creating Maintenance window "{}" for {} minutes' \
                    .format(MAINTENANCE_WINDOW_NAME + host, str(MAINTENANCE_WINDOW_DURATION_MINUTES)) \
                    , logTime=True)
                response = requests.post(
                    self.api_host + '/api/v1/customers/' + self.customer_uuid + '/maintenance_windows',
                    headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False,
                    json = j)
        else:
            if w_id is not None:
                log('\n- Removing Maintenance window "{}"'.format(MAINTENANCE_WINDOW_NAME + host), logTime=True)
                response = requests.delete(
                    self.api_host + '/api/v1/customers/' + self.customer_uuid + '/maintenance_windows/' + w_id,
                    headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False)
            else:
                log('\n- No existing Maintenance window "{}" found to remove' \
                    .format(MAINTENANCE_WINDOW_NAME + host), logTime=True)



    def find_window_by_name(self, host):
        name = MAINTENANCE_WINDOW_NAME + host
        response = requests.post(
            self.api_host + '/api/v1/customers/' + self.customer_uuid + '/maintenance_windows/list',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False,
            json={})
        for w in response.json():
            if w['name'] == name:
                return(w['uuid'])
        return None

    def get_customers(self):
            response = requests.get(self.api_host + '/api/customers/' + self.customer_uuid,
                                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token},
                                timeout=5, verify=False)
            response.raise_for_status()
            self.customers = response.json()

    def wait_for_task(self, task_uuid):
        jsonResponse = None
        response = requests.get(self.api_host + '/api/v1/customers/' + self.customer_uuid + '/tasks/' + task_uuid,
                                    headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token},verify=False)
        jsonResponse = response.json()
        if jsonResponse['status'] == 'Success':
            return True
        if jsonResponse['status'] == 'Failure':
            log('Task failed - see below for details', isError=True,logTime=True)
            log(json.dumps(jsonResponse, indent=2))
            return False
        raise ValueError("Still waiting for success/completion. Current state="+ jsonResponse['status'] + " at "+  str(jsonResponse['percent'])+"%")

    def alter_replication(self, xcluster_action, rpl_id):
        ## Get xcluster config
        response = requests.get(self.api_host + '/api/customers/' + self.customer_uuid + '/xcluster_configs/' + rpl_id,
                                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token},verify=False)
        xcl_cfg = response.json()
        # Now have 'paused = true/false' and status stays as running in yba 2.18
        xcNewState = 'Running'
        # Pause/resume as directed and if not in the correct state
        if xcluster_action == 'pause':
            xcNewState = 'Paused'

        xcCurrState = ''
        if 'paused' in xcl_cfg:
            if xcl_cfg['paused']:
                xcCurrState = 'Paused'
            else:
                xcCurrState = 'Running'
        else:
            xcCurrState = xcl_cfg['status']

        if xcCurrState != xcNewState:
            retries = 3
            while retries > 0:
                log( ' Changing state of xcluster replication ' + xcl_cfg['name'] + ' from ' + xcCurrState + ' to ' + xcNewState, logTime=True)
                response = requests.put(
                    self.api_host + '/api/customers/' + self.customer_uuid + '/xcluster_configs/' + rpl_id,
                    headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.api_token}, verify=False,
                    json={"status": xcNewState})
                if response.status_code == 200:
                    retries = 0
                    task = response.json()
                    if self.wait_for_task(task['taskUUID']):
                        log( ' Xcluster replication ' + xcl_cfg['name'] + ' is now ' + xcNewState, logTime=True)
                    else:
                        retries -= 1
                else:
                    retries -= 1
                    if retries > 0:
                        log('  XCluster task returned the following error {} - trying {} more times '.format(response.status_code, retries))
                        time.sleep(TASK_WAIT_TIME_SECONDS)
                    else:
                        log('  XCluster task returned the following error {} on final try - aborting '.format(response.status_code), True)
                        return False
        else:
            log('  Replication ' + xcl_cfg['name'] + ' already ' + xcNewState + ' - skipping')
        return True

#-------------------------------------------------------------------------------------------
class YB_Data_Node:

    def __init__(self,hostname,YBA_API,args):
        #log('Checking if host ' + hostname + ' is a YB Data node...')
        self.hostname = hostname
        self.ip       = get_node_ip(hostname)
        self.YBA_API  = YBA_API
        self.args     = args
        self.universe = None
        try: # See if the 'yugabyte' user exists
           services = subprocess.check_output(['id','-u','yugabyte'])
           self.yugabyte_id = int(services.decode())
        except:
            # Looks like the "yugabyte" user does not exist .. exit with exception
            raise NotMyTypeException("No user yugabyte")
        
        try:
           services = subprocess.check_output(["crontab","-u","yugabyte","-l"])
           if " master " in str(services)  or  " tserver " in str(services):
               return None # Successfuly instantiated
        except subprocess.CalledProcessError:
           pass # If the crontab is empty, it will have exit code 1, which we want to ignore

        try:           
           services = subprocess.check_output(['runuser','-l','yugabyte','-c','systemctl --user list-units --type=service --all'],stderr=subprocess.STDOUT)
           if 'yb-tserver.service' in str(services)  or  'yb-master.service' in str(services):
               return None
           # "master" / "tserver" were NOT FOUND 
           raise NotMyTypeException(hostname + " is not a YB Data node")
        except subprocess.CalledProcessError as e:
            log('supprocess Error checking for YB DB process: ', isError=True,logTime=True)
            log(e.output,isError=True)
            raise NotMyTypeException(hostname + " is not a YB Data node(No cron and Error getting services)")
        except Exception as e:
            log('Error checking for YB DB process: ', isError=True,logTime=True)
            log(e.output,isError=True)
            raise NotMyTypeException(hostname + " is not a YB Data node(No cron and Error getting services)")        

    @classmethod # Special constructor 
    def construct_from_json(cls,json,universe,YBA_API,args):
        self = cls.__new__(cls)  # Does not call __init__
        super(YB_Data_Node, self).__init__()  # call polymorphic base class initializers
        self.node_json = json
        self.universe  = universe 
        self.YBA_API   = YBA_API
        self.hostname = json['nodeName']
        self.ip       = json['cloudInfo']['private_ip']
        self.args     = args
        self.isMaster = json['isMaster']
        self.isTserver= json['isTserver']
        return self
    

    # Start node, then x-cluster - only print xluster status if dry run
    def resume(self): # aka start_node 
        self.YBA_API.Initialize()
        if self.universe is None:
            self.universe, self.node_json = self.YBA_API.find_universe_for_node(self.hostname, self.ip)        
        log('  Found node ' + self.node_json['nodeName'] + ' in Universe ' + self.universe.name
            + ' - UUID ' + self.universe.UUID)
        if self.args.dryrun:
            log('--- Dry run only - all checks will be done, but replication will not be resumed and nothing will be started ')
            log('Node ' + self.node_json['nodeName'] + ' state: ' + self.node_json['state'])
        else:
            ## Startup server
            log(' Starting up DB server', logTime=True)
            if self.node_json['state'] != 'Stopped':
                if self.node_json['state'] != 'Live':
                    log('  Node ' + self.node_json['nodeName'] + ' is in "' + self.node_json['state'] +
                        '" state - needs to be in "Stopped" or "Live" state to continue')
                    log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
                    exit(NODE_DB_ERROR)
                log('  Node ' + self.node_json['nodeName'] + ' is already in "Live" state - skipping startup')
            else:
                response = requests.put(
                    self.YBA_API.api_host + '/api/v1/customers/' + self.YBA_API.customer_uuid + '/universes/' + 
                        self.universe.UUID + '/nodes/' + self.node_json['nodeName'],
                    headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.YBA_API.api_token},
                    json=json.loads('{"nodeAction": "START"}'),verify=False)
                task = response.json()
                if 'error' in task:
                    log('Could not start node : ' + task['error'], logTime=True,isError=True)
                    log('Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
                    exit(NODE_DB_ERROR)
                if retry_successful(self.YBA_API.wait_for_task, params=[ task['taskUUID'] ],sleep=TASK_COMPLETE_WAIT_TIME_SECONDS,verbose=True,retry=15):
                    log(' Server startup complete', logTime=True)
                    restarted = True
                else:
                    raise Exception("Failed to resume DB Node")

        ## Resume x-cluster replication
        if self.args.skip_xcluster:
            log('\n- Skipping resume of x-cluster replication')
        else:
            self.universe.Resume_xCluster_replication()

        # Remove existing maintenence window
        if not self.args.skip_maint_window:
            self.YBA_API.maintenance_window( self, 'remove')

    def _compare_node_service_status_to_YBA (self,node_type):
        uri = None
        can_reach = True
        if node_type.lower() == 'master':
            uri = self.node_json['cloudInfo']['private_ip'] + ':' + str(self.node_json['masterHttpPort']) + '/api/v1/health-check'
        elif node_type.lower() == 'tserver':
            uri = self.node_json['cloudInfo']['private_ip'] + ':' + str(self.node_json['tserverHttpPort']) + '/api/v1/health-check'
        else:
            raise Exception('Invalid node type "{}" for node health'.format(node_type))

        try:
            resp = requests.get('http://' + uri)
        except:
            try:
                resp = requests.get('https://' + uri, verify=False)
            except:
                can_reach = False
                pass

        if self.node_json['state'] == 'Live':
            if can_reach:
                return True, None
            else:
                return False, '{} is running according to YBA, but stopped or uncommunicative'.format(node_type)
        else:
            if can_reach:
                return False, '{} is stopped according to YBA, but running on node'.format(node_type)
            else:
                return True, None


    # Verify tServer/Master processes are in same state as YBA thinks they are
    def verify(self):
        self.YBA_API.Initialize()
        if self.universe is None:
            self.universe, self.node_json = self.YBA_API.find_universe_for_node(self.hostname, self.ip)    
        log('Verifying Master and tServer on node {} are in correct state per YBA'.format(self.node_json['cloudInfo']['private_ip']))
        log(' - YBA shows node as being {}'.format(self.node_json['state']))
        errs = 0

        if self.node_json['state'] == 'Live':
            if self.node_json['isMaster']:
                log('   YBA shows node as having a Master - checking for process')
                passed, message = self._compare_node_service_status_to_YBA('Master')
                if passed:
                    log('     Check passed: master process found on node')
                else:
                    log(message, True)
                    errs += 1
            else:
                log('   YBA shows node as NOT having a Master - skipping check')

            if self.node_json['isTserver']:
                log('   YBA shows node as having a tServer - checking for process')
                passed, message = self._compare_node_service_status_to_YBA('tServer')
                if passed:
                    log('     Check passed: tServer process found on node')
                else:
                    log(message, True)
                    errs += 1
            else:
                log('   YBA shows node as NOT having a  tServer - skipping check')
        elif self.node_json['state'] == 'Stopped':
            passed, message = self._compare_node_service_status_to_YBA('Master')
            if passed:
                log('     Check passed: No master process found on node')
            else:
                log(message, True)
                errs += 1

            passed, message = self._compare_node_service_status_to_YBA('Tserver')
            if passed:
                log('     Check passed: No tServer process found on node')
            else:
                log(message, True)
                errs += 1
        else:
            log('Node is in state "' + self.node_json['state'] + '" and cannot be verified.  Node must be LIVE or STOPPED to run verification', True)
            errs += 1

        if errs > 0:
            raise Exception("Node process verification failed")
        return True

    # get Master leader
    def get_master_leader_ip(self):
        j = self.YBA_API.get_universe_info(self.universe.UUID,'/leader')
        #resp = requests.get(
        #    api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe['universeUUID'] + '/leader',
        #    headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        #j = resp.json()
        #print('---------- Master Leader debug ------------')
        #print(j)
        if not isinstance(j, dict):
            raise Exception("Call to get leader IP returned {} instead of dict".format(type(j)))
        if not 'privateIP' in j:
            raise Exception("Could not determine master leader - privateIP was not found in {}".format(j))
        return(get_node_ip(j['privateIP']))




    def stepdown_master(self, ldr_ip):
        status = subprocess.check_output(LEADER_STEP_DOWN_COMMAND.format(ldr_ip + ' master_leader_stepdown'),
                                                 shell=True, stderr=subprocess.STDOUT)
        time.sleep(random.randint(10, MAX_TIME_TO_SLEEP_SECONDS))
        new_ldr = self.get_master_leader_ip()
        if new_ldr == ldr_ip:
            raise Exception('   An error occurred while trying to step down master node  - proceeding with shutdown' )


    # Stop x-cluster and then the node processes
    def stop(self): #api_host, customer_uuid, universe, api_token, node, skip_xcluster=False, skip_maint_window=False):
        self.YBA_API.Initialize()
        if self.universe is None:
            self.universe, self.node_json = self.YBA_API.find_universe_for_node(self.hostname,self.ip)
        if self.universe.get_dead_node_count() > 0:
            log("Cannot stop node because one or more other nodes/services is down", isError=True)
            raise Exception("Cannot stop node because one or more other nodes/services is down")
        # Add maintenence window
        if not self.args.skip_maint_window:
            self.YBA_API.maintenance_window(self, 'create')

        ## Pause x-cluster replication if specified
        if self.args.skip_xcluster:
            log('\n- Skipping pause of x-cluster replication')
        else:
            self.universe.Pause_xCluster_Replication()

        ## Step down if master
        log('\n - Checking if node {} is master leader before shutting down'.format(self.node_json['cloudInfo']['private_ip']))
        ldr_ip = self.get_master_leader_ip()
        if ldr_ip == get_node_ip(self.node_json['cloudInfo']['private_ip']):
            if self.args.skip_stepdown:
                log("Skipping master-leader stepdown, as requested")
            else:
                log('   Node is currently master leader - stepping down before shutdown')
                if retry_successful(self.stepdown_master, params=[ldr_ip], verbose=True):
                    log('Master stepdown succeeded', logTime=True)
                    time.sleep(10) # Allow new master to read catalog etc
                else:
                    log('Failed to stepdown master', logTime=True)

        ## Shutdown server
        log(' Shutting down DB server ' + str(self.node_json['nodeName']), logTime=True)
        response = requests.put(
            self.YBA_API.api_host + '/api/v1/customers/' + self.YBA_API.customer_uuid + '/universes/' 
                          + self.universe.UUID + '/nodes/' + self.node_json['nodeName'],
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': self.YBA_API.api_token}, verify=False,
            json=json.loads('{"nodeAction": "STOP"}'),)
        task = response.json()
        if  task.get('taskUUID') is None:
            log("NODE STOP  task error:{}".format(task))
            raise Exception("Failed to create STOP NODE task")
        if 'error' in task:
            log('Could not shut down node : ' + task['error'], isError=True,logTime=True)
            log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
            exit(NODE_DB_ERROR)
        if retry_successful(self.YBA_API.wait_for_task, params=[ task['taskUUID'] ],sleep=TASK_COMPLETE_WAIT_TIME_SECONDS,verbose=True,retry=15):
            log(' Server shut down and ready for maintenance', logTime=True)
        else:
            log(' Error stopping node', True, logTime=True)
            raise Exception("Failed to stop Node")



    def health(self):
        self.YBA_API.Initialize()
        if self.universe is None:
            self.universe, self.node_json = self.YBA_API.find_universe_for_node(self.hostname,self.ip)
        if self.node_json is None:
            log("Node " + self.hostname + " is not in any known universe" ,isError=True,logTime=True)
            raise Exception("Node " + self.hostname + " is not in any known universe" )
        
        log('Found node ' + self.node_json['nodeName'] + ' in Universe ' + self.universe.name + ' - UUID ' + self.universe.UUID)
        self.universe.health_check()
        self.verify()

#-------------------------------------------------------------------------------------------

def Get_Environment_info():
    api_host      = os.environ.get("YBA_HOST")
    api_token     = os.environ.get("API_TOKEN")
    customer_uuid = os.environ.get("CUST_UUID")

    if api_host is not None  and  api_token is not None  and customer_uuid is not None:
        return (api_host,api_token,customer_uuid)
    if not os.path.exists(ENV_FILE_PATH):
        log(ENV_FILE_PATH + " does not exist.",isError=True)
        return (api_host,api_token,customer_uuid) # "None" values
    
    # find env variable file - should be only 1
    flist = fnmatch.filter(os.listdir(ENV_FILE_PATH), ENV_FILE_PATTERN)
    if len(flist) < 1:
        log('No environment variable file found in ' + ENV_FILE_PATH, isError=True,logTime=True)
        log('\nProcess failed - exiting with code ' + str(OTHER_ERROR))
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)
    if len(flist) > 1:
        log('Multiple environment variable files found in ' + ENV_FILE_PATH, True)
        log('Found the following files: ' + str(flist)[1:-1])
        log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)

    log('Retrieving environment variables from file ' + ENV_FILE_PATH + flist[0])
    env_file = open(ENV_FILE_PATH + flist[0], "r")
    ln = env_file.readline()
    while ln:
        if 'YBA_HOST' in ln:
            api_host = ln.split('=')[1].replace("'", "").replace('"', '').replace('\n', '').replace('\r', '')
        if 'API_TOKEN' in ln:
            api_token = ln.split('=')[1].replace("'", "").replace('"', '').replace('\n', '').replace('\r', '')
        if 'CUST_UUID' in ln:
            customer_uuid = ln.split('=')[1].replace("'", "").replace('"', '').replace('\n', '').replace('\r', '')
        ln = env_file.readline()
    env_file.close()
    missingEnv = False
    if api_host is None:
        log('Environment variable YBA_HOST not found', True)
        missingEnv = True
    if api_token is None:
        log('Environment variable API_TOKEN not found', True)
        missingEnv = True
    if customer_uuid is None:
        log('Environment variable CUST_UUID not found', True)
        missingEnv = True
    if missingEnv:
        log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)
    
    return (api_host,api_token,customer_uuid)

### Main Code ##############################################################################################
def main():
    ## parse the arguments
    parser = argparse.ArgumentParser(
        description='Yugabyte pre/post flight check - Start and Stop Services before and after O/S patch')
    mxgroup = parser.add_mutually_exclusive_group(required=True)
    mxgroup.add_argument('-s', '--stop',
                         action='store_true',
                         help='Stop services for YB host prior to O/S patch')
    mxgroup.add_argument('-r', '--resume',
                         action='store_true',
                         help='Resume services for YB host after O/S patch')
    mxgroup.add_argument('-t', '--health',
                         nargs='?',
                         const=LOCALHOST,
                         type=str,
                         action='store',
                         help='Healthcheck - specify Universe Name or "ALL" if not running on a DB Node')
    mxgroup.add_argument('-f', '--fix',
                         nargs='+',
                         action='store',
                         help='Fix one or more of the following: placement')
    mxgroup.add_argument('-v', '--verify',
                         action='store_true',
                         help='Verify Master and tServer process are in correct state per universe config')
    parser.add_argument('-d', '--dryrun',
                        action='store_true',
                        help='Dry Run - health checks only - no actions taken.',
                        required=False)
    parser.add_argument('-l', '--log_file_directory',
                        action='store',
                        help='Log file folder location.  Output is sent to terminal stdout if not specified.',
                        required=False)
    parser.add_argument('-x', '--skip_xcluster',
                        action='store_true',
                        help='Skip Pause or Resume of xCluster replication when stopping or resuming nodes - False if not specified, forced to True if stopping multiple nodes in a region/AZ.',
                        required=False,
                        default=False)
    parser.add_argument('-m', '--skip_maint_window',
                        action='store_true',
                        help='Skip creation/removal of maintenence window when stopping or resuming nodes - False if not specified, forced to True if stopping multiple nodes in a region/AZ.',
                        required=False,
                        default=False)
    parser.add_argument('-g', '--region',
                        action='store',
                        help='Region for nodes to be stopped/resumed - action taken on local node if not specified.',
                        required=False)
    parser.add_argument('-a', '--availability_zone',
                        action='store',
                        help='AZ for nodes to be stopped/resumed(optional) -  Script will abort if --region is not specified along with the AZ',
                        required=False)
    parser.add_argument('-e','--ENV_FILE_PATH',
                        action='store',
                        help='By default, the script will look for the ENV_FILE_PATTERN in /home/yugabyte. You can overwrite this by providing --ENV_FILE_PATH <new path>, example is --ENV_FILE_PATH /tmp/'

    )
    parser.add_argument('-u','--universe',
                        action='store',
                        help='Universe to operate on, or "ALL" (health, or regional ops)'
                        )
    parser.add_argument('-k', '--skip_stepdown',
                        action='store_true',
                        help='Skip master-stepdown if this is a STOP on a master-leader. If not set, we will attempt stepdown.',
                        required=False,
                        default=False)    
    args = parser.parse_args()

    hostname = str(socket.gethostname())
    ip = socket.gethostbyname(hostname)
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    requests.packages.urllib3.disable_warnings()
    dry_run = args.dryrun
    action = ''
    if args.health:
        action = 'health'
        dry_run = True
    elif args.stop:
        action = 'stop'
    elif args.resume:
        action = 'resume'
    elif args.fix:
        action = 'fix'
        dry_run = True
    elif args.verify:
        action = 'verify'
        dry_run = True
    # Overwritting the EVN_FILE_PATH
    if args.ENV_FILE_PATH is not None:
        global ENV_FILE_PATH
        ENV_FILE_PATH = args.ENV_FILE_PATH

    # Set up logging - if directory not specified, log to stdout
    if args.log_file_directory is not None:
        global LOG_TO_TERMINAL
        LOG_TO_TERMINAL = False
        global LOG_FILE
        logdir = str(args.log_file_directory)
        if not logdir.endswith('/'):
            logdir = logdir + '/'
        LOG_FILE = open(logdir + 'yb_os_maint_' + hostname + '_' + datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
                        + '_' + action + '.log', "a")

    log('\n--------------------------------------')
    log(' script version {} started on {} with action={}'.format(Version,hostname,action), logTime=True)
    if args.availability_zone is not None and args.region is None:
        log('--region parameter must be specified when --availability_zone is specified', True)
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)
    
    (api_host,api_token,customer_uuid) = Get_Environment_info()

    # ---- Mainline code -------
    YBA_API   = YBA_API_CLASS(api_host,customer_uuid,api_token,args) # Instantiated , but not Initialized yet
    this_node = None # The node object I will perform "action" upon
    for node_class in (Multiple_Nodes_Class, YBA_Node, YB_Data_Node):
        try:
            log ("Checking if node is of type '" + node_class.__name__ + "' ...",logTime=True)
            this_node = node_class(hostname,YBA_API,args)
            break # Found a matching class .. exit FOR loop
        except NotMyTypeException:
            continue # Try the next type
        except Exception as e:
            log ("Failed to instantiate "+ str(node_class) + " Exiting with code 6", isError=True,logTime=True)
            log(e,isError=True)
            exit (6)

    if this_node is None:
        log ("WARNING: Could not identify node type for "+hostname + ". Ignoring and terminating normally",logTime=True)
        exit (0)
        
    try:
       action_method = getattr(this_node,action) 
       action_method()  # instance "this_node" is already bound into action_method
       
    except AttributeError:
        log ("Could not perform '"+ action + "' because it is not valid for '" + str(this_node.__class__) + "'" , isError=True,logTime=True)
        exit(4)
    except Exception as e:
        log ("Failed "+ action + " Exiting with code 5", isError=True,logTime=True)
        log(e,isError=True)
        exit (5)
    else: 
        # We get here is there were NO exceptions
       log(' Process completed successfully - exiting with code ' + str(NODE_DB_SUCCESS), logTime=True)
       if(not LOG_TO_TERMINAL):
          LOG_FILE.close()
       exit (NODE_DB_SUCCESS)
    
#-------------------------------------------------------------------------------------------

if __name__ == '__main__':
    main()
