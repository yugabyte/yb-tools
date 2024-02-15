# coding: utf8
# !/usr/bin/python
###############################################################
## Application Control for use with UNIX Currency Automation ##
###############################################################

##  08/09/2022
##  Original Author: Mike LaSpina - Yugabyte

''' ---------------------- Change log ----------------------
V1.0 - Initial version
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
'''

Version = "1.34"

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
TASK_WAIT_TIME_SECONDS = 2
TASK_COMPLETE_WAIT_TIME_SECONDS = 10
YBA_PROCESS_STOP_LIST = ['yb-platform', 'prometheus', 'rh-nginx118-nginx', 'rh-postgresql10-postgresql']
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

# Check to see if we are on a YBA server by hitting up port 9090 - try both http and https
def yba_server(host, action, isDryRun):
    log('Checking if host ' + host + ' is YBA Instance for "' + action + '"...')
    found = False
    try:
        r = subprocess.check_output("systemctl list-units --all '{}.service'".format(YBA_PROCESS_STOP_LIST[0]), shell=True, stderr=subprocess.STDOUT)
        if '{}.service'.format(YBA_PROCESS_STOP_LIST[0]) in str(r):
            found = True

    except subprocess.CalledProcessError as e:
        log('Error checking for YBA process: ', isError=True,logTime=True)
        log(e.output)

    if found:
        if action == "test":
            return (True) # Caller just checking if this is a YBA 
        activePath = subprocess.check_output(['readlink','-f','/opt/yugabyte/software/active'],text=True)
        ybaVersion = activePath.rstrip('\n').split("/")[-1] # Last dir is a version like '2.18.5.2-b1'
        if isDryRun:
            log('Host is YBA Server {} - Dry run or Health check specified:no action taken'.format(ybaVersion),logTime=True)
        else:
            if action == 'stop':
                log(' Host is YBA Server {} - Shutting down services...'.format(ybaVersion), logTime=True)
                if ybaVersion >= '2.18.0':
                    try:
                        status=subprocess.check_output(['yba-ctl','stop'],  stderr=subprocess.STDOUT) # No output
                        time.sleep(2)
                        status=subprocess.check_output(['yba-ctl','status'],stderr=subprocess.STDOUT,text=True) 
                        log(status,logTime=True)
                        return(True)
                    except subprocess.CalledProcessError as e:
                        log('  yba-ctl stop failed - skipping. Err:{}'.format(str(e)),logTime=True)
                else:    
                  for svc in YBA_PROCESS_STOP_LIST:
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

            if action == 'resume':
                log(' Host is YBA Server - Starting up services...', logTime=True)
                if ybaVersion >= '2.18.0':
                    try:
                        status=subprocess.check_output(['yba-ctl','start'],  stderr=subprocess.STDOUT) # No output
                        time.sleep(2)
                        status=subprocess.check_output(['yba-ctl','status'], stderr=subprocess.STDOUT,text=True)
                        log(status,logTime=True)
                        return(True)
                    except subprocess.CalledProcessError as e:
                        log('  yba-ctl start failed. Err:{}'.format(str(e)),logTime=True,isError=True)
                        exit(NODE_YBA_ERROR)
                for svc in reversed(YBA_PROCESS_STOP_LIST):
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
        return(True)
    else:
        if action == 'test':
            return (False)
        log('  Host is NOT YBA Instance - checking if it is a DB Node')
        return (False)


# Get all universes on YBA deployment.  Note that a list of nodes is included here, so we return the entire universes array
def get_universes(api_host, customer_uuid, api_token):
    response = requests.get(api_host + '/api/customers/' + customer_uuid + '/universes',
                            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token}, verify=False)
    return (response.json())

def maintenance_window(api_host, customer_uuid, universe, api_token, host, action):
    w_id = find_window_by_name(api_host, customer_uuid, api_token, host)
    if action == 'create':
        mins_to_add = timedelta(minutes=MAINTENANCE_WINDOW_DURATION_MINUTES)
        j = {"customerUUID" : customer_uuid,
             "name" : MAINTENANCE_WINDOW_NAME + host,
             "description" : MAINTENANCE_WINDOW_NAME + host,
             "startTime": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
             "endTime": (datetime.now() + mins_to_add).strftime("%Y-%m-%d %H:%M:%S"),
             "alertConfigurationFilter": {
                 "targetType": "UNIVERSE",
                 "target": {
                     "all": False,
                     "uuids": [universe['universeUUID']]
                 }
             }
        }
        if w_id is not None:
            log('\n- Updating existing Maintenance window "{}" for {} minutes' \
                .format(MAINTENANCE_WINDOW_NAME + host, str(MAINTENANCE_WINDOW_DURATION_MINUTES)) \
                , logTime=True)
            j['uuid'] = w_id
            response = requests.put(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows/' + w_id,
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json = j)
        else:
            log('\n- Creating Maintenance window "{}" for {} minutes' \
                .format(MAINTENANCE_WINDOW_NAME + host, str(MAINTENANCE_WINDOW_DURATION_MINUTES)) \
                , logTime=True)
            response = requests.post(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows',
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json = j)
    else:
        if w_id is not None:
            log('\n- Removing Maintenance window "{}"'.format(MAINTENANCE_WINDOW_NAME + host), logTime=True)
            response = requests.delete(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows/' + w_id,
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        else:
            log('\n- No existing Maintenance window "{}" found to remove' \
                .format(MAINTENANCE_WINDOW_NAME + host), logTime=True)



def find_window_by_name(api_host, customer_uuid, api_token, host):
    name = MAINTENANCE_WINDOW_NAME + host
    response = requests.post(
        api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows/list',
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
        json={})
    for w in response.json():
        if w['name'] == name:
            return(w['uuid'])
    return None

# Start node, then x-cluster - only print xluster status if dry run
def start_node(api_host, customer_uuid, universe, api_token, node, dry_run=True, skip_xcluster=False, skip_maint_window=False):
    try:
        log('  Found node ' + node['nodeName'] + ' in Universe ' + universe['name'] + ' - UUID ' + universe[
            'universeUUID'])
        if dry_run:
            log('--- Dry run only - all checks will be done, but replication will not be resumed and nothing will be started ')
            log('Node ' + node['nodeName'] + ' state: ' + node['state'])
        else:
            ## Startup server
            log(' Starting up DB server', logTime=True)
            if node['state'] != 'Stopped':
                if node['state'] != 'Live':
                    log('  Node ' + node['nodeName'] + ' is in "' + node['state'] +
                        '" state - needs to be in "Stopped" or "Live" state to continue')
                    log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
                    exit(NODE_DB_ERROR)
                log('  Node ' + node['nodeName'] + ' is already in "Live" state - skipping startup')
            else:
                response = requests.put(
                    api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe[
                        'universeUUID'] + '/nodes/' + node['nodeName'],
                    headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                    json=json.loads('{"nodeAction": "START"}'))
                task = response.json()
                if 'error' in task:
                    log('Could not start node : ' + task['error'], logTime=True,isError=True)
                    log('Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
                    exit(NODE_DB_ERROR)
                if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
                    log(' Server startup complete', logTime=True)
                    restarted = True
                else:
                    raise Exception("Failed to resume DB Node")

        ## Resume x-cluster replication
        if skip_xcluster:
            log('\n- Skipping resume of x-cluster replication')
        else:
            ## resume source replication
            log('\n- Resuming x-cluster replication')
            resume_count = 0
            if 'sourceXClusterConfigs' in universe['universeDetails']:
                for rpl in universe['universeDetails']['sourceXClusterConfigs']:
                    if dry_run:
                        response = requests.get(
                            api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl,
                            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
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
                        if alter_replication(api_host, api_token, customer_uuid, 'resume', rpl):
                            resume_count += 1
                        else:
                            raise Exception("Failed to resume x-cluster replication")

            ## resume target replication
            if 'targetXClusterConfigs' in universe['universeDetails']:
                for rpl in universe['universeDetails']['targetXClusterConfigs']:
                    if dry_run:
                        response = requests.get(
                            api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl,
                            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
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
                        if alter_replication(api_host, api_token, customer_uuid, 'resume', rpl):
                            resume_count += 1
                        else:
                            raise Exception("Failed to resume x-cluster replication")
            if resume_count > 0:
                if dry_run:
                    log('  ' + str(resume_count) + ' x-cluster streams were found, but not resumed due to dry run')
                else:
                    log('  ' + str(resume_count) + ' x-cluster streams are now running')
            else:
                log('  No x-cluster replications were found to resume')


        # Remove existing maintenence window
        if not skip_maint_window:
            maintenance_window(api_host, customer_uuid, universe, api_token, node['nodeName'], 'remove')

    except Exception as e:
        log(e, True)
        log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
        exit(NODE_DB_ERROR)

def get_node_health (node_type, node, yba_state):
    uri = None
    can_reach = True
    if node_type.lower() == 'master':
        uri = node['cloudInfo']['private_ip'] + ':' + str(node['masterHttpPort']) + '/api/v1/health-check'
    elif node_type.lower() == 'tserver':
        uri = node['cloudInfo']['private_ip'] + ':' + str(node['tserverHttpPort']) + '/api/v1/health-check'
    else:
        raise Exception('Invalid node type "{}" for node health'.format(node_type))

    try:
        resp = requests.get('http://' + uri)
    except:
        try:
            resp = requests.get('https://' + uri)
        except:
            can_reach = False
            pass

    if yba_state == 'Live':
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
def verify(api_host, customer_uuid, universe, api_token, node):
    log('Verifying Master and tServer on node {} are in correct state per YBA'.format(node['cloudInfo']['private_ip']))
    log(' - YBA shows node as being {}'.format(node['state']))
    errs = 0

    if node['state'] == 'Live':
        if node['isMaster']:
            log('   YBA shows node as having a Master - checking for process')
            passed, message = get_node_health('Master', node, node['state'])
            if passed:
                log('     Check passed: master process found on node')
            else:
                log(message, True)
                errs += 1
        else:
            log('   YBA shows node as NOT having a Master - skipping check')

        if node['isTserver']:
            log('   YBA shows node as having a tServer - checking for process')
            passed, message = get_node_health('tServer', node, node['state'])
            if passed:
                log('     Check passed: tServer process found on node')
            else:
                log(message, True)
                errs += 1
        else:
            log('   YBA shows node as NOT having a  tServer - skipping check')
    elif node['state'] == 'Stopped':
        passed, message = get_node_health('Master', node, node['state'])
        if passed:
            log('     Check passed: No master process found on node')
        else:
            log(message, True)
            errs += 1

        passed, message = get_node_health('Tserver', node, node['state'])
        if passed:
            log('     Check passed: No tServer process found on node')
        else:
            log(message, True)
            errs += 1
    else:
        log('Node is in state "' + node['state'] + '" and cannot be verified.  Node must be LIVE or STOPPED to run verification', True)
        errs += 1

    if errs > 0:
        raise Exception("Node process verification failed")
    return True

# get Master leader
def get_master_leader_ip(api_host, customer_uuid, api_token, universe):
    resp = requests.get(
        api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe['universeUUID'] + '/leader',
        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
    j = resp.json()
    print('----------------------')
    print(j)
    if not isinstance(j, dict):
        raise Exception("Call to get leader IP returned {} instead of dict".format(type(j)))
    if not 'privateIP' in j:
        raise Exception("Could not determine master leader - privateIP was not found in {}".format(j))
    return(get_node_ip(j['privateIP']))


def get_node_ip(addr):
    try:
        socket.inet_aton(addr)
        return(addr)
    except:
        return(socket.gethostbyname(addr))

def retry_successful(retriable_function_call, params=None, retry=10, verbose=False, sleep=.5, fatal=False):
    for i in range(retry):
        try:
            verbose and log("Attempt {} running {}".format(i, retriable_function_call.__name__),logTime=True)
            time.sleep(sleep * i)
            retval = retriable_function_call(*params)
            verbose and log(
                "  Returned {} from called function {} on attempt {}".format(retval, retriable_function_call.__name__, i))
            return True
        except Exception as errorMsg:
            preserve_errmsg = errorMsg
            verbose and log("Hit exception {} in attempt {}".format(errorMsg, i))
            if fatal and i == (retry - 1):
                raise  # Re-raise current exception
            continue

    return False

def stepdown_master(api_host, customer_uuid, api_token, universe, node, ldr_ip):
    status = subprocess.check_output(LEADER_STEP_DOWN_COMMAND.format(ldr_ip + ' master_leader_stepdown'),
                                             shell=True, stderr=subprocess.STDOUT)
    time.sleep(random.randint(10, MAX_TIME_TO_SLEEP_SECONDS))
    new_ldr = get_master_leader_ip(api_host, customer_uuid, api_token, universe)
    if new_ldr == ldr_ip:
        raise Exception('   An error occurred while trying to step down master node  - proceeding with shutdown' )


# Stop x-cluster and then the node processes
def stop_node(api_host, customer_uuid, universe, api_token, node, skip_xcluster=False, skip_maint_window=False):
    try:
        # Add maintenence window
        if not skip_maint_window:
            maintenance_window(api_host, customer_uuid, universe, api_token, node['nodeName'], 'create')

        ## Pause x-cluster replication if specified
        if skip_xcluster:
            log('\n- Skipping pause of x-cluster replication')
        else:
            ## pause source replication
            log('\n- Pausing x-cluster replication', logTime=True)
            paused_count = 0
            ## pause source replication
            if 'sourceXClusterConfigs' in universe['universeDetails']:
                for rpl in universe['universeDetails']['sourceXClusterConfigs']:
                    ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
                    time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
                    if alter_replication(api_host, api_token, customer_uuid, 'pause', rpl):
                        paused_count += 1
                    else:
                        raise Exception("Failed to pause x-cluster replication")
            ## pause target replication
            if 'targetXClusterConfigs' in universe['universeDetails']:
                for rpl in universe['universeDetails']['targetXClusterConfigs']:
                    ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
                    time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
                    if alter_replication(api_host, api_token, customer_uuid, 'pause', rpl):
                        paused_count += 1
                    else:
                        raise Exception("Failed to pause x-cluster replication")
            if paused_count > 0:
                log('  ' + str(paused_count) + ' x-cluster streams are currently paused', logTime=True)
            else:
                log('  No x-cluster replications were found to pause', logTime=True)

        ## Step down if master
        log('\n - Checking if node {} is master leader before shutting down'.format(node['cloudInfo']['private_ip']))
        ldr_ip = get_master_leader_ip(api_host, customer_uuid, api_token, universe)
        if ldr_ip == get_node_ip(node['cloudInfo']['private_ip']):
            log('   Node is currently master leader - stepping down before shutdown')
            if retry_successful(stepdown_master, params=[api_host, customer_uuid, api_token, universe, node, ldr_ip], verbose=True):
                log('Master stepdown succeeded', logTime=True)
            else:
                log('Failed to stepdown master', logTime=True)

        ## Shutdown server
        log(' Shutting down DB server' + str(node['nodeName']), logTime=True)
        response = requests.put(
            api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe[
                'universeUUID'] + '/nodes/' + node['nodeName'],
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
            json=json.loads('{"nodeAction": "STOP"}'))
        task = response.json()
        if 'error' in task:
            log('Could not shut down node : ' + task['error'], True)
            log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
            exit(NODE_DB_ERROR)
        if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
            log(' Server shut down and ready for maintenance', logTime=True)
        else:
            log(' Error stopping node', True, logTime=True)
            raise Exception("Failed to stop Node")
    except Exception as e:
        log(e, True)
        log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
        exit(NODE_DB_ERROR)


def health_check(api_host, customer_uuid, universe, api_token, node=None):
    try:
        errcount = 0;
        log('Performing health checks for universe {}, UUID {}...'.format(universe['name'], universe['universeUUID']))

        # put together Prometheus URL by stripping off existing port of API server if specified and appending proper port
        # Then try https, if fails, step down to http
        tmp_url = api_host.split(':')
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
                log('Could not contact prometheus host at {}'.format(promhost), True)
                errcount += 1;
        log('Using prometheus host at {}\n'.format(promhost))

        if node is not None:
            log('Found node ' + node['nodeName'] + ' in Universe ' + universe['name'] + ' - UUID ' + universe['universeUUID'])

        log('- Checking for task placement UUID')
        if (PLACEMENT_TASK_FIELD in universe and len(universe[PLACEMENT_TASK_FIELD]) > 0) or \
                (PLACEMENT_TASK_FIELD in universe['universeDetails'] and len(universe['universeDetails'][PLACEMENT_TASK_FIELD]) > 0):
            log('Non-empty ' + PLACEMENT_TASK_FIELD + ' found in universe', True)
            errcount +=1
        else:
            log('  Passed placement task test')

        log('- Checking for dead nodes, tservers and masters')
        resp = requests.get(
            api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe['universeUUID'] + '/status',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        nlist = resp.json()
        # this one comes back as a dict instead of proper JSON, also last key is uuid, which we need to ignore
        for n in nlist:
            if n != 'universe_uuid':
                if nlist[n]['node_status'] != 'Live':
                    log('  Node ' + n + ' is not alive', True)
                    errcount += 1

                for nodes in universe['universeDetails']['nodeDetailsSet']:
                    if nodes['nodeName'] == n:
                        if nodes['isMaster'] and not nlist[n]['master_alive']:
                            log('  Node ' + n + ' master is not alive', True)
                            errcount += 1

                        if nodes['isTserver'] and not nlist[n]['tserver_alive']:
                            log('  Node ' + n + ' tserver is not alive', True)
                            errcount += 1
        if errcount == 0:
            log('  All nodes, masters and t-servers are alive.')

        log('- Checking for master and tablet health')
        if node is not None:
            if not node['isMaster']:
                log('  ### Warning: node is not a Master ###')
            if not node['isTserver']:
                log('  ### Warning: node is not a TServer ###')

        master_node = None
        for masternode in universe['universeDetails']['nodeDetailsSet']:
            if masternode['isMaster'] and masternode['state'] == 'Live':
                master_node = masternode['cloudInfo']['private_ip'] + ':' + str(masternode['masterHttpPort'])
                break

        try:  # try both http and https endpoints
            resp = requests.get('https://' + master_node + '/api/v1/health-check')
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
                        for tnode in universe['universeDetails']['nodeDetailsSet']:
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
            resp = requests.get('https://' + master_node + '/api/v1/tablet-servers')
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
            htmlresp = requests.get('https://' + master_node + '/tablet-servers')
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
            for tnode in universe['universeDetails']['nodeDetailsSet']:
                if tnode['isTserver']:
                    promnodes += tnode['nodeName'] + '|'
            promql = 'max(max_over_time(follower_lag_ms{exported_instance=~"' + promnodes +\
                     '",export_type="tserver_export"}[' + str(LAG_LOOKBACK_MINUTES) + 'm]))'
            log('  Executing the following Prometheus query:')
            log('   ' + promql)
            resp = requests.get(promhost, params={'query':promql}, verify=False)
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
        resp = requests.get(
            api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe['universeUUID'] + '/masters',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        master_list = resp.json()
        num_masters = len(str(master_list).split(','))
        if universe['universeDetails']['clusters'][0]['userIntent']['replicationFactor'] == num_masters:
            log('  Check passed - cluster has RF of {} and {} masters'.format(
                universe['universeDetails']['clusters'][0]['userIntent']['replicationFactor'],
                num_masters))
        else:
            log('Check failed - cluster has RF of {} and {} masters'.format(
                universe['universeDetails']['clusters'][0]['userIntent']['replicationFactor'],
                num_masters), True)
            errcount+=1

        # Check master lag
        if totalTablets > 0:
            log('  Checking replication lag for masters')
            promnodes =''
            for mnode in universe['universeDetails']['nodeDetailsSet']:
                if mnode['isMaster']:
                    promnodes += mnode['nodeName'] + '|'
            promql = 'max(max_over_time(follower_lag_ms{exported_instance=~"' + promnodes +\
                     '",export_type="master_export"}[' + str(LAG_LOOKBACK_MINUTES) + 'm]))'
            log('  Executing the following Prometheus query:')
            log('   ' + promql)
            resp = requests.get(promhost, params={'query':promql}, verify=False)
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
                errcount) + ' errors were detected terminating shutdown.')
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
            exit(NODE_DB_ERROR)
        else:
            log('--- Health check for universe "{}" completed with no issues'.format(universe['name']))
            return

    except Exception as e:
        log(e, True)
        log(' Process failed - exiting with code ' + str(NODE_DB_ERROR), logTime=True)
        exit(NODE_DB_ERROR)

def fix(api_host, customer_uuid, universe, api_token, fix_list):
    log('Fixing the following items in the universe: ' + str(fix_list))
    mods = []
    f = universe['universeDetails']
    if 'placement' in fix_list:
        if PLACEMENT_TASK_FIELD in f and len(f[PLACEMENT_TASK_FIELD]) > 0:
            f[PLACEMENT_TASK_FIELD] = ''
            mods.append('placement')

    if mods:
        log('Updating universe config with the following fixed items: ' + str(mods))
        if 'tserverGFlags' not in f:
            f['tserverGFlags'] = {"vmodule": "secure1*"}
        if 'masterGFlags' not in f:
            f['masterGFlags'] = {"vmodule": "secure1*"}
        f['upgradeOption'] = "Non-Restart"
        f['sleepAfterMasterRestartMillis'] = 0
        f['sleepAfterTServerRestartMillis'] = 0
        #f['kubernetesUpgradeSupported'] = False
        response = requests.post(api_host + '/api/v1/customers/' + customer_uuid + '/universes/' +
                                 universe['universeUUID'] + '/upgrade/gflags',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
            json=f)
        task = response.json()
        if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
            log('Server items fixed')
        else:
            log('Fix task failed', True)
    else:
        log('No items exist to fix')

def get_db_nodes_and_universe(universes, hostname, ip, universe_name, region=None, az=None):
    if universes is None or len(universes) < 1:
        log('No Universes found - cannot determine if this a DB node. **EXITING NORMALLY**', isError=True, logTime=True)
        #log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
        #raise Exception("No Universes found")
        exit(0)
    univ_to_return = None
    nodes = []
    curnode = None

    # First, get universe
    for universe in universes:
        if universe_name == LOCALHOST:
            for node in universe['universeDetails']['nodeDetailsSet']:
                if str(node['nodeName']).upper() in hostname.upper() or hostname.upper() in \
                    str(node['nodeName']).upper() or \
                        node['cloudInfo']['private_ip'] == ip or \
                        node['cloudInfo']['public_ip'] == ip or \
                        node['cloudInfo']['private_ip'].upper() in hostname.upper():
                    univ_to_return = universe
                    curnode = node
                    break
        if universe_name != LOCALHOST and universe_name != 'ALL':
            if str(universe['name']).upper() == str(universe_name).upper():
                univ_to_return = universe
                break
   
    # Get node(s) for given universe
    if region is not None:
        for node in univ_to_return['universeDetails']['nodeDetailsSet']:
            if node['cloudInfo']['region'].upper() == region.upper():
                if az is not None:
                    if node['cloudInfo']['az'].upper() == az.upper():
                        nodes.append(node)
                else:
                    nodes.append(node)
    else:
        nodes.append(curnode)

    return nodes, univ_to_return

def wait_for_task(api_host, customer_uuid, task_uuid, api_token):
    done = False
    jsonResponse = None
    while not done:
        time.sleep(TASK_WAIT_TIME_SECONDS)
        response = requests.get(api_host + '/api/v1/customers/' + customer_uuid + '/tasks/' + task_uuid,
                                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        jsonResponse = response.json()
        if jsonResponse['status'] == 'Failure' or jsonResponse['percent'] == 100.0:
            done = True
        if jsonResponse['status'] == 'Running':
            done = False

    if jsonResponse['status'] != 'Success':
        log('Task failed - see below for details', isError=True,logTime=True)
        log(json.dumps(jsonResponse, indent=2))
        return False
    else:
        time.sleep(TASK_COMPLETE_WAIT_TIME_SECONDS)
        return True

def alter_replication(api_host, api_token, customer_uuid, xcluster_action, rpl_id):
    ## Get xcluster config
    response = requests.get(api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl_id,
                            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
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
                api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl_id,
                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json={"status": xcNewState})
            if response.status_code == 200:
                retries = 0
                task = response.json()
                if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
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

### Main Code
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
                         help='Healthcheck only - specify Universe Name or "ALL" if not running on a DB Node')
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
                        help='AZ for nodes to be stopped/resumed - action taken on local node if not specified.  Script will abort if --region is not specified along with the AZ',
                        required=False)
    args = parser.parse_args()

    hostname = str(socket.gethostname())
    ip = socket.gethostbyname(hostname)
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
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

    ACTIONS_ALLOWED_ON_YBA = 'health|stop|resume'

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
    log(' script version {} started'.format(Version), logTime=True)
    if args.availability_zone is not None and args.region is None:
        log('--region parameter must be specified when --availability_zone is specified', True)
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)
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
    api_host = None
    api_token = None
    customer_uuid = None
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

    if action == 'resume' and yba_server(hostname,'test',True):
        # YBA host is down, so NO API is available -- just restart it...
        yba_server(hostname,action,dry_run)
        log(' Resume completed successfully - exiting with code ' + str(NODE_DB_SUCCESS), logTime=True)
        exit(0) # No furthur processing .. YBA processes are not fully up yet.

    num_retries = 1
    while num_retries <= 5:
        try:
            log('Attempt {} to communicate with YBA host at {}'.format(num_retries, api_host))
            response = requests.get(api_host + '/api/customers/' + customer_uuid,
                                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                                timeout=5, verify=False)
            response.raise_for_status()
            num_retries = 999
        except requests.exceptions.HTTPError as err:
            if num_retries >= 5:
                log(' Could not establish communication with YBA server at {} after {} attempts Err {}- exiting'.format(api_host, num_retries,err),isError=True, logTime=True)
                log(' Process failed - exiting with code {}'.format(NODE_YBA_ERROR), logTime=True)
                if (not LOG_TO_TERMINAL):
                    LOG_FILE.close()
                exit(NODE_YBA_ERROR)
            else:
                num_retries += 1
                sleeptime = random.randint(5, MAX_TIME_TO_SLEEP_SECONDS)
                log('Could not establish communication with YBA server at {} trying again in {} seconds. Err: {}'.format(api_host, sleeptime,err), logTime=True)
                pass

    log('Retrieving Universes from YBA server at ' + api_host)
    universes = get_universes(api_host, customer_uuid, api_token)
    rg = None
    az = None
    univ_name = LOCALHOST
    if action == 'health':
        univ_name = args.health
    if action == 'stop' or action == 'resume':
        rg = args.region
        az = args.availability_zone
    dbhost_list, universe = get_db_nodes_and_universe(universes, hostname, ip, univ_name, rg, az)
    if universe == None:
        if yba_server(hostname,'test',dry_run):
            dbhost_list=[] # Zap the incorrect [None] value 
            pass #no op
        else:
            log("Did not find any universe for host {} IP {}. Ignoring unknown host and *EXITING NORMALLY*".format(hostname,ip),
                 isError=True,logTime=True)
            exit(0)
    try:
        ## first, do healthcheck if specified
        if action == 'health':
            log('--- Health Check only - all checks will be done, but nothing will be stopped or resumed ')

            if args.health == LOCALHOST and len(dbhost_list) < 1:
                log('Healthcheck is not being run from a DB node - Specify a universe name or "ALL" to check all Universes from a non-DB node.', True)
                log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
                if (not LOG_TO_TERMINAL):
                    LOG_FILE.close()
                exit(OTHER_ERROR)
            if args.health != LOCALHOST:
                if len(dbhost_list) < 1:
                    log('Healthcheck universe name specified from a DB node - checking health for universe "{}" instead of universe associated with the node'.format(args.health))
                if str(args.health).upper() == 'ALL':
                    for hc_universe in universes:
                        log('\n')
                        health_check(api_host, customer_uuid, hc_universe, api_token, None)
                else:
                    if universe is not None:
                        health_check(api_host, customer_uuid, universe, api_token, None)
                    else:
                        log('Could not find universe with name "{}" exiting'.format(args.health), isError=True)
                        log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
                        if (not LOG_TO_TERMINAL):
                            LOG_FILE.close()
                        exit(OTHER_ERROR)
            else:
                dbhost = None
                if len(dbhost_list) > 0:
                    dbhost = dbhost_list[0]
                health_check(api_host, customer_uuid, universe, api_token, dbhost)

        else: # not a healthckeck only, so proceed
            if len(dbhost_list) < 1: # running from YBA instance
                if yba_server(ip, action, dry_run):
                    exit_code = NODE_YBA_SUCCESS
                    if not action in ACTIONS_ALLOWED_ON_YBA :
                        log('Cannot run {} command from a YBA server - run from the node instead'.format(action), isError=True)
                        log(' Process failed - exiting with code ' + str(NODE_YBA_ERROR), logTime=True)
                        exit_code = NODE_YBA_ERROR
                    else:
                        log(' Process completed successfully - exiting with code ' + str(NODE_YBA_SUCCESS), logTime=True)
                    if (not LOG_TO_TERMINAL):
                        LOG_FILE.close()
                    exit(exit_code)
                else: # not YBA or DB node, and not running a healthcheck
                    log(' Node is neither a DB host nor a YBA host - exiting with code ' + str(OTHER_ERROR), logTime=True)
                    if (not LOG_TO_TERMINAL):
                        LOG_FILE.close()
                    exit(OTHER_ERROR)
            else: # running from a DBnode or region specified
                skip_xc = args.skip_xcluster
                skip_maint = args.skip_maint_window
                node_list_text = ''
                for dbhost in dbhost_list:
                    node_list_text = node_list_text + str(dbhost['nodeName']) + ', '
                node_list_text = node_list_text.rstrip(', ')
                if len(dbhost_list) > 1:
                    skip_xc = True
                    skip_maint = True
                if action == 'stop':
                    log(' The following nodes were found to stop: {}\n'.format(node_list_text))
                    if dry_run:
                        log('--- Dry run only - all checks will be done, but replication will not be paused and nothing will be stopped ')
                    health_check(api_host, customer_uuid, universe, api_token, dbhost_list[0])
                    if dry_run:
                        log('--- Dry run only - Exiting')
                    else:
                        for dbhost in dbhost_list:
                            stop_node(api_host, customer_uuid, universe, api_token, dbhost, skip_xc, skip_maint)
                elif action == 'resume':
                    log(' The following nodes were found to resume: {}\n'.format(node_list_text))
                    if dry_run:
                        log('--- Dry run only - Exiting')
                    else:
                        for dbhost in dbhost_list:
                            start_node(api_host, customer_uuid, universe, api_token, dbhost, dry_run, skip_xc, skip_maint)
                elif action == 'fix':
                    fix(api_host, customer_uuid, universe, api_token, args.fix)
                elif action == 'verify':
                    for dbhost in dbhost_list:
                        verify(api_host, customer_uuid, universe, api_token, dbhost)
    except Exception as e:
        log(e, True)
        log(' Process failed - exiting with code ' + str(OTHER_ERROR), logTime=True)
        if(not LOG_TO_TERMINAL):
            LOG_FILE.close()
        #traceback.print_exc()
        exit(OTHER_ERROR)

    log(' Process completed successfully - exiting with code ' + str(NODE_DB_SUCCESS), logTime=True)
    if(not LOG_TO_TERMINAL):
        LOG_FILE.close()

if __name__ == '__main__':
    main()
