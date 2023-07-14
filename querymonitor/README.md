# querymonitor
![querymonitor icon](querymonitor_icon.png "Querymonitor")
## Synopsis
Runs (as a daemon) and periodically collects live query info into mime-encoded csv (.gz) files.
Personal Identifiable Information (PII) can be removed from each query by truncating the WHERE clause.
Built-in `--ANALYZE` mode will analyze the generated file.

The data collected is suitable for offline analysis to obtain:
* Statement level response times AKA slow queries
* Node response times & volumes
* system behaviour by time
* SLO compliance

## Installation / Environment
This requires perl, and a few core modules that should be present in any Linux system.
It obtains live query info from YBA, and requires parameters and connectivity :
* `API_TOKEN`
* `YBA_HOST` 
* `CUST_UUID`
* `UNIV_UUID`

These can be supplied via environment variables, the command line, or a flag file.

By default, the program runs in the background, collecting query samples every 5 seconds, for 4 hours, then exits.

## Run instructions

If you have the environment variable set, you can simply run the code with no parameters:

`./querymonitor.pl`
  
You can  create a flag file containing the required params like
`myuniv_queryparms.flag`

```
--API_TOKEN=<value>
--YBA_HOST =<value>
--CUST_UUID=<value>
--UNIV_UUID=<value>
```

You can specify the flagfile to use:

`./querymonitor.pl --flagfile=myuniv_queryparms.flag`

If the program finds a file called `querymonitor.defaultflags`, 
the contents are automatically loaded unless the `--flagfile` option is used.

A `--help` parameter provides a full list of available flags.

```
$ ./querymonitor.pl --help
2023-04-17 13:45 Starting ./querymonitor.pl version 0.05  PID 6235 on LAPTOP-5976NRBG
#    querymonitor.pl  Version 0.05
#    ===============
# Monitor running queries
# collect gzipped JSON file for offline analysis

Program Options:     Parameter(if-any)     Description
  --ANALYZE     <file-name>           Analyzes the file and creates a Sqlite DB + reports
  --API_TOKEN   <token-UUID>          From "User Profile"
  --CURL        <path-to-curl-binary> 
  --CUST_UUID   <UUID>                 From "User Profile" 
  --DAEMON                             Run as Daemon (ON by default. use --NODAEMON for test)
  --DB        <Path to sqlite DB t be created> In --ANALYZE mode only
  --DEBUG                              Prints verbose messages      
  --FLAGFILE   <Path-to-flag-file>     Reads runtime flags from disk. Default is querymonitor.defaultflags     
  --HELP                               Shows these flags 
  --HTTPCONNECT  <connect-method>      Defaults to "curl". Other option is 'http' (HTTP::Tiny)           
  --INTERVAL_SEC  <Default is 5s (seconds)> How often to poll for  queries 
  --MAX_QUERY_LEN                     Truncates queries longer than this
  --OUTPUT   <file-name>              Name of output (mime) file 
  --RPCZ                              Polls nodes directly. (--norpcz polls the /live_queries endpoint)
  --RUN_FOR    <4h by default>        Auto-terminate after this time 
  --SANITIZE                          Whther to truncate queries at WHERE clause  
  --SQLITE     <path to sqlite binary>                       
  --UNIV_UUID                         From YBA Universe page
  --USETESTDATA                       Testing only   
  --VERSION                           Show version 
  --YBA_HOST    <URL to connect to YBA>                      REQUIRED 
```

## Analysis mode

After data has been collected by this program, the file can also be processed, using the `--ANALYZE` option.
This will de-compress and process the file through SQLITE, and generate reports.
Here is a sample run:

```
$ perl querymonitor.pl  --analyze queries.ycql.XX.2023-04-24.csv.gz
2023-04-24 15:39 Starting querymonitor.pl version 0.08  PID 1781 on MYHOSTNAME
Analyzing queries.ycql.XX.2023-04-24.csv.gz ...
Creating temporary fifo /tmp/querymonitor_fifo_1781 ...
Creating sqlite database queries.ycql.XX.2023-04-24.sqlite ...
SQLite 3.37.2 2022-01-06 13:25:41 872ba256cbf61d9290b571c0e6d82a20c224ca3ad82971edc46b29818d5dalt1
zlib version 1.2.11
gcc-11.3.0
Imported 8301 rows from queries.ycql.XX.2023-04-24.csv.gz.

====== Summary Report ====
EDT                  systemq  cqlcount  sys_gt120  cql_gt120  sys_dc3  cql_dc3  breach_pct
-------------------  -------  --------  ---------  ---------  -------  -------  ----------
2023-04-24 10:40:00  0        45        0          0          0        0        0.0
2023-04-24 10:50:00  2        376       0          2          2        0        0.53
2023-04-24 11:00:00  0        272       0          3          0        0        1.1
2023-04-24 11:10:00  0        270       0          1          0        0        0.37
2023-04-24 11:20:00  0        335       0          0          0        0        0.0
2023-04-24 11:30:00  1        269       0          0          0        0        0.0
2023-04-24 11:40:00  0        255       0          0          0        0        0.0
2023-04-24 11:50:00  0        305       0          1          0        0        0.33
2023-04-24 12:00:00  191      255       132        0          126      0        0.0
2023-04-24 12:10:00  0        225       0          0          0        0        0.0
2023-04-24 12:20:00  88       249       84         0          47       0        0.0
2023-04-24 12:30:00  87       242       0          1          65       0        0.41
2023-04-24 12:40:00  1        234       0          0          0        0        0.0
2023-04-24 12:50:00  85       235       74         0          58       0        0.0
2023-04-24 13:00:00  52       277       29         0          40       0        0.0
2023-04-24 13:10:00  0        256       0          0          0        0        0.0
2023-04-24 13:20:00  0        302       0          1          0        0        0.33
2023-04-24 13:30:00  218      296       102        0          125      0        0.0
2023-04-24 13:40:00  0        272       0          2          0        0        0.74
2023-04-24 13:50:00  2        311       0          0          1        0        0.0
2023-04-24 14:00:00  448      289       64         0          309      0        0.0
2023-04-24 14:10:00  51       286       24         0          32       0        0.0
2023-04-24 14:20:00  86       236       0          0          72       0        0.0
2023-04-24 14:30:00  414      263       44         1          279      0        0.38
2023-04-24 14:40:00  0        220       0          0          0        0        0.0

======= Slow Queries =======
query                                                                                                                                                                       nbr_querys  avg_milli  pct_gt120  DC3_querys
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------  ----------  ---------  ---------  ----------
SELECT * FROM system.peers                                                                                                                                                  230         119.5      35         171
SELECT * FROM system.local WHERE key='local'                                                                                                                                219         116.0      37         155
SELECT keyspace_name, table_name, start_key, end_key, replica_addresses FROM system.partitions                                                                              1277        98.2       30         830
select customerid,emails,individual407typecode,rescountrycode,customertypecode,inccountrycode,emailremindertimestamp,isemailunsolicited from customer where customerid = ?  783         44.4       1          743
```
