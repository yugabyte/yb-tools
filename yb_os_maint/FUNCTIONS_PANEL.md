# Functions Panel - yb_os_maint.py

## Global Functions
- **`log(msg, isError=False, logTime=False, newline=False)`** (Line 206)
  - Log messages to output - all messages go through this function
  
- **`retry_successful(retriable_function_call, params=None, retry=10, verbose=False, sleep=0.5, fatal=False, ReturnFuncVal=False)`** (Line 222)
  - Generic retry logic for function calls with configurable retry attempts
  
- **`get_node_ip(addr)`** (Line 250)
  - Returns dotted-decimal address, handles both IP and hostname
  
- **`Get_Environment_info()`** (Line 1691)
  - Retrieves environment information from configuration files
  
- **`main()`** (Line 1755)
  - Main entry point for the script, handles argument parsing and execution flow

## Universe Class Methods
- **`__init__(self, YBA_API, json)`** (Line 265)
  - Constructor for Universe class
  
- **`Populate_NodeObjectList(self)`** (Line 280)
  - Populates the nodeObjectList with YB_Data_Node objects
  
- **`get_node_json(self, hostname, ip=None)`** (Line 308)
  - Finds node JSON by hostname or IP
  
- **`find_nodes_by_region_az(self, region, az)`** (Line 325)
  - Finds nodes by region and availability zone
  
- **`get_master_leader_ip(self, include_port=False)`** (Line 338)
  - Gets master leader IP address
  
- **`check_under_replicated_tablets(self)`** (Line 348)
  - Checks if system has under-replicated tablets
  
- **`get_dead_node_count(self, log_it=True, exclude_ip=None)`** (Line 391)
  - Counts dead nodes in the universe
  
- **`get_health_info_from_master(self, master_ip_and_port)`** (Line 414)
  - Retrieves health information from master node
  
- **`follower_lag_exceeded(self, server_type, threshold_seconds, label_dimensions="")`** (Line 427)
  - Checks if follower lag exceeds threshold
  
- **`health_check(self)`** (Line 476)
  - Performs comprehensive health checks for the universe
  
- **`fix(self)`** (Line 653)
  - Fixes issues in the universe
  
- **`Pause_xCluster_Replication(self)`** (Line 699)
  - Pauses xCluster replication
  
- **`Resume_xCluster_replication(self)`** (Line 726)
  - Resumes xCluster replication

## YBA Node Class Methods
- **`__init__(self, host, YBA_API, args)`** (Line 788)
  - Constructor for YBA node class
  
- **`health(self)`** (Line 825)
  - Performs health checks for YBA node
  
- **`fix(self)`** (Line 841)
  - Fixes issues for YBA node
  
- **`stop(self)`** (Line 844)
  - Stops YBA node services
  
- **`resume(self)`** (Line 882)
  - Resumes YBA node services

## YB Database Node Class Methods
- **`__init__(self, host, YBA_API, args)`** (Line 904)
  - Constructor for YB database node class
  
- **`setVersion(self)`** (Line 928)
  - Sets version information
  
- **`health(self)`** (Line 932)
  - Performs health checks for database node
  
- **`resume(self)`** (Line 943)
  - Resumes database node services
  
- **`stop(self)`** (Line 975)
  - Stops database node services
  
- **`fix(self)`** (Line 1005)
  - Fixes issues for database node

## YBA API Class Methods
- **`__init__(self, env_dict, args)`** (Line 1011)
  - Constructor for YBA API class
  
- **`Initialize(self)`** (Line 1020)
  - Initializes the YBA API connection
  
- **`prometheus_request(self, querystr)`** (Line 1025)
  - Makes Prometheus metric requests
  
- **`_Initialize_w_retry(self)`** (Line 1036)
  - Initializes with retry logic
  
- **`find_universe_by_name_or_uuid(self, lookfor_name=None)`** (Line 1068)
  - Finds universe by name or UUID
  
- **`find_universe_by_region_az(self, region, az)`** (Line 1078)
  - Finds universe by region and availability zone
  
- **`find_universe_for_node(self, hostname=None, ip=None)`** (Line 1106)
  - Finds universe for a specific node
  
- **`get_universe_info(self, univ_uuid, endpoint)`** (Line 1115)
  - Gets universe information from API
  
- **`snooze_health_alerts(self, universe=None, disable=True, duration_sec=3600)`** (Line 1121)
  - Snoozes health alerts
  
- **`active_alerts(self, universe)`** (Line 1132)
  - Gets active alerts for universe
  
- **`maintenance_window(self, node, action)`** (Line 1153)
  - Manages maintenance windows
  
- **`search_maintenance_windows(self, host=None, callback=None)`** (Line 1213)
  - Searches for maintenance windows
  
- **`delete_expired_maintenance_windows(self)`** (Line 1231)
  - Deletes expired maintenance windows
  
- **`get_customers(self)`** (Line 1251)
  - Gets customer information
  
- **`wait_for_task(self, task_uuid)`** (Line 1258)
  - Waits for task completion
  
- **`alter_replication(self, xcluster_action, rpl_id)`** (Line 1272)
  - Alters xCluster replication

## YB Data Node Class Methods
- **`__init__(self, hostname, YBA_API, args)`** (Line 1322)
  - Constructor for YB data node class
  
- **`construct_from_json(cls, json, universe, YBA_API, args)`** (Line 1361)
  - Class method to construct from JSON
  
- **`__eq__(self, other)`** (Line 1378)
  - Equality comparison method
  
- **`node_action_api_call(self, action)`** (Line 1383)
  - Makes API calls for node actions
  
- **`Print_node_info_line(self)`** (Line 1390)
  - Prints node information line
  
- **`resume(self)`** (Line 1406)
  - Resumes node services
  
- **`_compare_node_service_status_to_YBA(self, node_type)`** (Line 1459)
  - Compares node service status to YBA
  
- **`verify(self)`** (Line 1491)
  - Verifies node configuration
  
- **`get_master_leader_ip(self)`** (Line 1545)
  - Gets master leader IP for this node
  
- **`stepdown_master(self, ldr_ip)`** (Line 1562)
  - Steps down master node
  
- **`stop(self)`** (Line 1572)
  - Stops node services
  
- **`health(self)`** (Line 1650)
  - Performs health checks for node
  
- **`reprovision(self)`** (Line 1662)
  - Reprovisions the node

## Usage Examples

### Health Check
```bash
python yb_os_maint.py -t [universe_name|ALL]
```

### Stop Services
```bash
python yb_os_maint.py -s [--region region_name] [--availability_zone az_name]
```

### Resume Services
```bash
python yb_os_maint.py -r [--region region_name] [--availability_zone az_name]
```

### Fix Issues
```bash
python yb_os_maint.py -f placement
```

### Verify Configuration
```bash
python yb_os_maint.py -v
```

### Dry Run
```bash
python yb_os_maint.py -s -d  # Stop with dry run
```

## Key Features
- **Multi-node Operations**: Support for stopping/resuming nodes by region and availability zone
- **Health Monitoring**: Comprehensive health checks for universes and nodes
- **xCluster Management**: Automatic pause/resume of xCluster replication
- **Maintenance Windows**: Creation and management of maintenance windows
- **Prometheus Integration**: Metric-based health monitoring
- **Retry Logic**: Robust error handling with configurable retry attempts
- **Logging**: Comprehensive logging with configurable output destinations
