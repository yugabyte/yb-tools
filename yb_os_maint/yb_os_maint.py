# coding: utf8
# !/usr/bin/python
###############################################################
## Application Control for use with UNIX Currency Automation ##
###############################################################

##  08/09/2022
##  Mike LaSpina - Yugabyte
##  mlaspina@yugabyte.com

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
    Added tablet balance check and basic underreplicated chacks
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
'''

Version = 1.18

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
MAINTENANCE_WINDOW_NAME = "OS Patching - "
MAINTENANCE_WINDOW_DURATION_MINUTES = 20

# Global scope variables - do not change!
LOG_TO_TERMINAL = True
LOG_FILE = None

## Custom Functions

# Log messages to output - all messages go thru this function
def log(msg, isError=False):
    if isError:
        msg = '*** Error ***: ' + str(msg)
    if LOG_TO_TERMINAL:
        print(msg)
    else:
        LOG_FILE.write(msg + '\n')

# Check to see if we are on a YBA server by hitting up port 9090 - try both http and https
def yba_server(host, action, isDryRun):
    log('Checking if host ' + host + ' is YBA Instance...')
    found = False
    try:
        r = subprocess.check_output("systemctl list-units --all '{}.service'".format(YBA_PROCESS_STOP_LIST[0]), shell=True, stderr=subprocess.STDOUT)
        if '{}.service'.format(YBA_PROCESS_STOP_LIST[0]) in str(r):
            found = True
    except subprocess.CalledProcessError as e:
        log('Error checking for process: ', True)
        log(e.output)

    if found:
        if isDryRun:
            log('Host is YBA Server - Dry run or Health check specified:no action taken')
        else:
            if action == 'stop':
                log(datetime.now().strftime(LOG_TIME_FORMAT) + ' Host is YBA Server - Shutting down services...')
                for svc in YBA_PROCESS_STOP_LIST:
                    try:
                        # This call triggers an error if the process is not active.
                        log('  Stopping service ' + svc)
                        status = subprocess.check_output('systemctl is-active {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                        try:
                            o = subprocess.check_output('systemctl stop {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                            log(datetime.now().strftime(LOG_TIME_FORMAT) + '  Service ' + svc + ' stopped')
                        except subprocess.CalledProcessError as err:
                            log('Error stopping service' + svc + '- see output below', True)
                            log(err)
                            log('\nProcess failed - exiting with code ' + str(NODE_YBA_ERROR))
                            exit(NODE_YBA_ERROR)
                    except subprocess.CalledProcessError as e:
                        log('  Service {} is not running - skipping'.format(svc))

            if action == 'resume':
                log(datetime.now().strftime(LOG_TIME_FORMAT) + ' Host is YBA Server - Starting up services...')
                for svc in reversed(YBA_PROCESS_STOP_LIST):
                    try:
                        # This call triggers an error if the process is not active.
                        log('  Stopping service ' + svc)
                        status = subprocess.check_output('systemctl is-active {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                        log('  Service {} is already running - skipping'.format(svc))
                    except subprocess.CalledProcessError as e: # This is the code path if the service is not running
                        try:
                            o = subprocess.check_output('systemctl start {}.service'.format(svc), shell=True, stderr=subprocess.STDOUT)
                            log(datetime.now().strftime(LOG_TIME_FORMAT) + '  Service ' + svc + ' started')
                        except subprocess.CalledProcessError as err:
                            log('Error stopping service' + svc + '- see output below', True)
                            log(err)
                            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_YBA_ERROR))
                            exit(NODE_YBA_ERROR)
        return(True)
    else:
        log('  Host is NOT YBA Instance - checking if it is a DB Node')
        return (False)


# Get all universes on YBA deployment.  Note that a list of nodes is included here, so we return the entire universes array
def get_universes(api_host, customer_uuid, api_token):
    response = requests.get(api_host + '/api/customers/' + customer_uuid + '/universes',
                            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
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
            log('\n- Updating existing Maintenance window "{}" for {} minutes'.format(MAINTENANCE_WINDOW_NAME + host,
                                                                           str(MAINTENANCE_WINDOW_DURATION_MINUTES)))
            j['uuid'] = w_id
            response = requests.put(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows/' + w_id,
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json = j)
        else:
            log('\n- Creating Maintenance window "{}" for {} minutes'.format(MAINTENANCE_WINDOW_NAME + host,
                                                                           str(MAINTENANCE_WINDOW_DURATION_MINUTES)))
            response = requests.post(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows',
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json = j)
    else:
        if w_id is not None:
            log('\n- Removing Maintenance window "{}"'.format(MAINTENANCE_WINDOW_NAME + host))
            response = requests.delete(
                api_host + '/api/v1/customers/' + customer_uuid + '/maintenance_windows/' + w_id,
                headers = {'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
        else:
            log('\n- No existing Maintenance window "{}" found to remove'.format(MAINTENANCE_WINDOW_NAME + host))



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
def start_node(api_host, customer_uuid, universe, api_token, node, dry_run=True):
    try:
        log('Found node ' + node['nodeName'] + ' in Universe ' + universe['name'] + ' - UUID ' + universe[
            'universeUUID'])
        if dry_run:
            log('--- Dry run only - all checks will be done, but replication will not be resumed and nothing will be started ')
            log('Node ' + node['nodeName'] + ' state: ' + node['state'])
        else:
            ## Startup server
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Starting up DB server')
            if node['state'] != 'Stopped':
                if node['state'] != 'Live':
                    log('  Node ' + node['nodeName'] + ' is in "' + node['state'] +
                        '" state - needs to be in "Stopped" or "Live" state to continue')
                    log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
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
                    log('Could not start node : ' + task['error'], True)
                    log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
                    exit(NODE_DB_ERROR)
                if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
                    log(datetime.now().strftime(LOG_TIME_FORMAT) + ' Server startup complete')
                    restarted = True
                else:
                    raise Exception("Failed to resume DB Node")

        ## Resume x-cluster replication
        ## resume source replication
        log('\n- Resuming x-cluster replication')
        ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
        time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
        resume_count = 0
        if 'sourceXClusterConfigs' in universe['universeDetails']:
            for rpl in universe['universeDetails']['sourceXClusterConfigs']:
                if dry_run:
                    response = requests.get(
                        api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl,
                        headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token})
                    xcl_cfg = response.json()
                    log('  Replication ' + xcl_cfg['name'] + ' is in state ' + xcl_cfg['status'])
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
                    log('  Replication ' + xcl_cfg['name'] + ' is in state ' + xcl_cfg['status'])
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
            log('No x-cluster replications were found to resume')

        # Remove existing maintenence window
        maintenance_window(api_host, customer_uuid, universe, api_token, node['nodeName'], 'remove')
        
    except Exception as e:
        log(e, True)
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
        exit(NODE_DB_ERROR)


# Stop x-cluster and then the node processes
def stop_node(api_host, customer_uuid, universe, api_token, node):
    try:
        # Add maintenence window
        maintenance_window(api_host, customer_uuid, universe, api_token, node['nodeName'], 'create')

        ## Pause x-cluster replication
        ## pause source replication
        log('\n- Pausing x-cluster replication')
        ## First, sleep a bit to prevent race condition when patching multiple servers concurrently
        time.sleep(random.randint(1, MAX_TIME_TO_SLEEP_SECONDS))
        paused_count = 0
        ## pause source replication
        if 'sourceXClusterConfigs' in universe['universeDetails']:
            for rpl in universe['universeDetails']['sourceXClusterConfigs']:
                if alter_replication(api_host, api_token, customer_uuid, 'pause', rpl):
                    paused_count += 1
                else:
                    raise Exception("Failed to pause x-cluster replication")

        ## pause target replication
        if 'targetXClusterConfigs' in universe['universeDetails']:
            for rpl in universe['universeDetails']['targetXClusterConfigs']:
                if alter_replication(api_host, api_token, customer_uuid, 'pause', rpl):
                    paused_count += 1
                else:
                    raise Exception("Failed to pause x-cluster replication")
        if paused_count > 0:
            log('  ' + str(paused_count) + ' x-cluster streams are currently paused')
        else:
            log('No x-cluster replications were found to pause')

        ## Shutdown server
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) +  ' Shutting down DB server')
        response = requests.put(
            api_host + '/api/v1/customers/' + customer_uuid + '/universes/' + universe[
                'universeUUID'] + '/nodes/' + node['nodeName'],
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
            json=json.loads('{"nodeAction": "STOP"}'))
        task = response.json()
        if 'error' in task:
            log('Could not shut down node : ' + task['error'], True)
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
            exit(NODE_DB_ERROR)
        if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
            log(datetime.now().strftime(LOG_TIME_FORMAT) + ' Server shut down and ready for maintenance')
        else:
            log(datetime.now().strftime(LOG_TIME_FORMAT) + ' Error stopping node', True)
            raise Exception("Failed to stop Node")
    except Exception as e:
        log(e, True)
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
        exit(NODE_DB_ERROR)


# Run pre-flight checks
def health_check(api_host, customer_uuid, universe, api_token, node):
    try:
        errcount = 0;
        log('Performing pre-flight checks...')

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

        log('Found node ' + node['nodeName'] + ' in Universe ' + universe['name'] + ' - UUID ' + universe[
            'universeUUID'])

        log('- Checking for task placement UUID')
        if PLACEMENT_TASK_FIELD in universe and len(universe[PLACEMENT_TASK_FIELD]) > 0:
            log('Non-empty ' + PLACEMENT_TASK_FIELD + ' found in universe', True)
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
        if not node['isMaster']:
            log('  ### Warning: node is not a Master ###')
        if not node['isTserver']:
            log('  ### Warning: node is not a TServer ###')

        master_node = None
        for masternode in universe['universeDetails']['nodeDetailsSet']:
            if masternode['isMaster'] and masternode['state'] == 'Live':
                master_node = masternode['cloudInfo']['private_ip']
                break

        try:  # try both http and https endpoints
            resp = requests.get('https://' + master_node + ':7000/api/v1/health-check')
            hc = resp.json()
        except:
            resp = requests.get('http://' + master_node + ':7000/api/v1/health-check')
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
            resp = requests.get('https://' + master_node + ':7000/api/v1/tablet-servers')
            tabs = resp.json()
        except:
            resp = requests.get('http://' + master_node + ':7000/api/v1/tablet-servers')
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
            htmlresp = requests.get('https://' + master_node + ':7000/tablet-servers')
        except:
            htmlresp = requests.get('http://' + master_node + ':7000/tablet-servers')

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

        ## End pre-flight checks,
        if errcount > 0:
            log('--- Pre flight check has failed - ' + str(
                errcount) + ' errors were detected terminating shutdown.')
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
            exit(NODE_DB_ERROR)
        else:
            log('--- Pre flight check completed with no issues')
            return

    except Exception as e:
        log(e, True)
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_DB_ERROR))
        exit(NODE_DB_ERROR)

def fix(api_host, customer_uuid, universe, api_token, dbhost, fix_list):
    log('Fixing the following items in the universe: ' + str(fix_list))
    mods = []
    f = copy.deepcopy(universe)
    if 'placement' in fix_list:
        if PLACEMENT_TASK_FIELD in f and len(f[PLACEMENT_TASK_FIELD]) > 0:
            f[PLACEMENT_TASK_FIELD] = ''
            mods.append('placeemnt')

    # Do not allow master/tserver fix for now.
    """
    if 'tserver' in fix_list or 'master' in fix_list:
        i = 0
        found = False
        # find correct node to fix
        while not found:
            if dbhost['nodeUuid'] == f['universeDetails']['nodeDetailsSet'][i]['nodeUuid']:
                found = True
            else:
                i+=1

        if 'master' in fix_list:
            if not f['universeDetails']['nodeDetailsSet'][i]['isMaster']:
                f['universeDetails']['nodeDetailsSet'][i]['isMaster'] = True
                mods.append('master')

        if 'tserver' in fix_list:
            if not f['universeDetails']['nodeDetailsSet'][i]['isTserver']:
                f['universeDetails']['nodeDetailsSet'][i]['isTserver'] = True
                mods.append('tserver')
    """

    if mods:
        log('Updating universe config with the following fixed items: ' + str(mods))
        response = requests.post(api_host + '/api/v1/customers/' + customer_uuid + '/universes/' +
                                 universe['universeUUID'] + '/upgrade/gflags',
            headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
            json=f)
        task = response.json()
        if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
            log('Server items fixed')
    else:
        log('No items exist to fix')

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

    if jsonResponse['status'] != 'Success':
        log('Task failed - see below for details', True)
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

    xcNewState = 'Running'
    # Pause/resume as directed and if not in the correct state
    if xcluster_action == 'pause':
        xcNewState = 'Paused'

    if xcl_cfg['status'] != xcNewState:
        retries = 3
        while retries > 0:
            log('  ' + datetime.now().strftime(LOG_TIME_FORMAT) +
                ' Changing state of xcluster replication ' + xcl_cfg['name'] + ' to ' + xcNewState)
            response = requests.put(
                api_host + '/api/customers/' + customer_uuid + '/xcluster_configs/' + rpl_id,
                headers={'Content-Type': 'application/json', 'X-AUTH-YW-API-TOKEN': api_token},
                json={"status": xcNewState})
            if response.status_code == 200:
                retries = 0
                task = response.json()
                if wait_for_task(api_host, customer_uuid, task['taskUUID'], api_token):
                    log('  ' + datetime.now().strftime(LOG_TIME_FORMAT) +
                        ' Xcluster replication ' + xcl_cfg['name'] + ' is now ' + xcNewState)
                else:
                    return False
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
                         action='store_true',
                         help='Healthcheck only - forces Dry Run')
    mxgroup.add_argument('-f', '--fix',
                         nargs='+',
                         action='store',
                         help='Fix one or more of the following: placement')
    parser.add_argument('-d', '--dryrun',
                        action='store_true',
                        help='Dry Run - pre-flight checks only - no actions taken.',
                        required=False)
    parser.add_argument('-l', '--log_file_directory',
                        action='store',
                        help='Log file folder location.  Output is sent to terminal stdout if not specified.',
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
    log(datetime.now().strftime(LOG_TIME_FORMAT) + ' script started')
    # find env variable file - should be only 1
    flist = fnmatch.filter(os.listdir(ENV_FILE_PATH), ENV_FILE_PATTERN)
    if len(flist) < 1:
        log('No environment variable file found in ' + ENV_FILE_PATH, True)
        log('\nProcess failed - exiting with code ' + str(OTHER_ERROR))
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)
    if len(flist) > 1:
        log('Multiple environment variable files found in ' + ENV_FILE_PATH, True)
        log('Found the following files: ' + str(flist)[1:-1])
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(OTHER_ERROR))
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
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(OTHER_ERROR))
        if (not LOG_TO_TERMINAL):
            LOG_FILE.close()
        exit(OTHER_ERROR)

    try:
        dbhost = None
        log('Checking host ' + hostname + ' with IP ' + ip)

        # determine instance type - DB Node, YBA Server, or other
        try:
            if yba_server(ip, action, dry_run):
                exit_code = NODE_YBA_SUCCESS
                if action == 'fix':
                    log('Cannot run fix command from a YBA server - run from the node instead', True)
                    log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(NODE_YBA_ERROR))
                    exit_code = NODE_YBA_ERROR
                else:
                    log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process completed successfully - exiting with code ' + str(NODE_YBA_SUCCESS))
                if (not LOG_TO_TERMINAL):
                    LOG_FILE.close()
                exit(exit_code)
        except Exception as e:
            log(e, True)
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(OTHER_ERROR))
            if (not LOG_TO_TERMINAL):
                LOG_FILE.close()
            exit(OTHER_ERROR)
        try:
            can_connect = True
            log('Checking current node using YBA server at ' + api_host)
        except:
            log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Could not contact YBA server at ' + api_host + ' exiting', True)
            exit(OTHER_ERROR)

        if can_connect:
            universes = get_universes(api_host, customer_uuid, api_token)

            # Iterate thru universes and attempt a match in the node list by node name, private IP or public IP
            # Note that node info is located in nodeDetailSet, so we only need to call getUniverses API to get all nodes
            if universes is None or len(universes) < 1:
                log('No Universes found - cannot determine if this a DB node', True)
                log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(OTHER_ERROR))
                exit(OTHER_ERROR)
            for universe in universes:
                for node in universe['universeDetails']['nodeDetailsSet']:
                    if str(node['nodeName']).upper() in hostname.upper() or hostname.upper() in str(
                            node['nodeName']).upper() or \
                            node['cloudInfo']['private_ip'] == ip or node['cloudInfo']['public_ip'] == ip:
                        dbhost = node
                        found = True
                        if action == 'stop':
                            if dry_run:
                                log('--- Dry run only - all checks will be done, but replication will not be paused and nothing will be stopped ')
                            health_check(api_host, customer_uuid, universe, api_token, dbhost)
                            if not dry_run:
                                stop_node(api_host, customer_uuid, universe, api_token, dbhost)
                            else:
                                log('--- Dry run only - Exiting')
                        elif action == 'resume':
                            start_node(api_host, customer_uuid, universe, api_token, dbhost, dry_run)
                        elif action == 'health':
                            log('--- Health Check only - all checks will be done, but replication will not be paused and nothing will be stopped ')
                            health_check(api_host, customer_uuid, universe, api_token, dbhost)
                        elif action == 'fix':
                            fix(api_host, customer_uuid, universe, api_token, dbhost, args.fix)

            if dbhost is None:
                log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Node is neither a DB host nor a YBA host - exiting with code ' + str(OTHER_SUCCESS))
                if (not LOG_TO_TERMINAL):
                    LOG_FILE.close()
                exit(OTHER_SUCCESS)
    except Exception as e:
        log(e, True)
        log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process failed - exiting with code ' + str(OTHER_ERROR))
        if(not LOG_TO_TERMINAL):
            LOG_FILE.close()
        #traceback.print_exc()
        exit(OTHER_ERROR)

    log('\n' + datetime.now().strftime(LOG_TIME_FORMAT) + ' Process completed successfully - exiting with code ' + str(NODE_DB_SUCCESS))
    if(not LOG_TO_TERMINAL):
        LOG_FILE.close()

if __name__ == '__main__':
    main()