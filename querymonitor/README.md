# querymonitor

## Synopsis
Runs (as a daemon) and periodically collects live query info into mime-encoded csv (.gz) files.
Personably Identifyable information (PII) can be removed from each query by truncating the WHERE clause.
Built-in --ANALYZE mode will analyze the generated file.

The data collected is suitable for offline analysis to obtain:
* Statement level response times AKA slow queries
* Node response times & volumes
* system behaviour by time
* SLO compliance

## Installation / Environment
This requires perl, and a few core modules that should be present in any Linux system.
It obtains live query info from YBA, and requires parameters and connectivity :
* API_TOKEN
* YBA_HOST 
* CUST_UUID
* UNIV_UUID

These can be supplied via environment variables, the command line, or a flag file.

By default, the program runs in the background, collecting query samples every 5 seconds, for 4 hours, then exits.

## Run instructions

If you have the environment variable set, you can simply run the code with no parameters:

<CODE>  ./querymonitor.pl</CODE>
  
You can  create a flag file containing the required params like
<code>myuniv_queryparms.flag</code>

<code>
--API_TOKEN= &lt;value><br>
--YBA_HOST = &lt;value><br>
--CUST_UUID= &lt;value><BR>
--UNIV_UUID= &lt;value>
</code>

You can specify the flagfile to use:

<code>
  ./querymonitor.pl --flagfile=myuniv_queryparms.flag
</code>

If the program finds a file called <code>querymonitor.defaultflags</code>, 
the contents are automatically loaded unless the <code>--flagfile</code> option is used.

A <code>--help</code> parameter provides a full list of available flags.

<code>
$ ./querymonitor.pl --help
<br/>2023-04-17 13:45 Starting ./querymonitor.pl version 0.05  PID 6235 on LAPTOP-5976NRBG
<br/>#    querymonitor.pl  Version 0.05
<br/>#    ===============
<br/># Monitor running queries
<br/># collect gzipped JSON file for offline analysis
<br/>
Program Options:     Parameter(if-any)     Description
<br/>  --ANALYZE     <file-name>           Analyzes the file and creates a Sqlite DB + reports
<br/>  --API_TOKEN   <token-UUID>          From "User Profile"
<br/>  --CURL        <path-to-curl-binary> 
<br/>  --CUST_UUID   <UUID>                 From "User Profile" 
<br/>  --DAEMON                             Run as Daemon (ON by default. use --NODAEMON for test)
<br/>  --DB        <Path to sqlite DB t be created> In --ANALYZE mode only
<br/>  --DEBUG                              Prints verbose messages      
<br/>  --FLAGFILE   <Path-to-flag-file>     Reads runtime flags from disk. Default is querymonitor.defaultflags     
<br/>  --HELP                               Shows these flags 
<br/>  --HTTPCONNECT  <connect-method>      Defaults to "curl". Other option is 'http' (HTTP::Tiny)           
<br/>  --INTERVAL_SEC  <Default is 5s (seconds)> How often to poll for  queries 
<br/>  --MAX_QUERY_LEN                     Truncates queries longer than this
<br/>  --OUTPUT   <file-name>              Name of output (mime) file 
<br/>  --RPCZ                              Polls nodes directly. (--norpcz polls the /live_queries endpoint)
<br/>  --RUN_FOR    <4h by default>        Auto-terminate after this time 
<br/>  --SANITIZE                          Whther to truncate queries at WHERE clause  
<br/>  --SQLITE     <path to sqlite binary>                       
<br/> --UNIV_UUID                         From YBA Universe page
<br/> --USETESTDATA                       Testing only   
<br/>  --VERSION                           Show version 
<br/>  --YBA_HOST    <URL to connect to YBA>                      REQUIRED 
</code>

## Analysis mode

After data has been collected by this program, the file can also be processed, using the --analyze option.
This will de-compress and process the file through SQLITE, and generate reports.
Here is a sample run:

<code>
$ perl querymonitor.pl  --analyze queries.ycql.XX.2023-04-24.csv.gz
<br/>2023-04-24 15:39 Starting querymonitor.pl version 0.08  PID 1781 on MYHOSTNAME
<br/>Analyzing queries.ycql.XX.2023-04-24.csv.gz ...
<br/>Creating temporary fifo /tmp/querymonitor_fifo_1781 ...
<br/>Creating sqlite database queries.ycql.XX.2023-04-24.sqlite ...
<br/>SQLite 3.37.2 2022-01-06 13:25:41 872ba256cbf61d9290b571c0e6d82a20c224ca3ad82971edc46b29818d5dalt1
<br/>zlib version 1.2.11
<br/>gcc-11.3.0
<br/>Imported 8301 rows from queries.ycql.XX.2023-04-24.csv.gz.
<br/>
<br/>====== Summary Report ====
<br/>EDT                  systemq  cqlcount  sys_gt120  cql_gt120  sys_dc3  cql_dc3  breach_pct
<br/>-------------------  -------  --------  ---------  ---------  -------  -------  ----------
<br/>2023-04-24 10:40:00  0        45        0          0          0        0        0.0
<br/>2023-04-24 10:50:00  2        376       0          2          2        0        0.53
<br/>2023-04-24 11:00:00  0        272       0          3          0        0        1.1
<br/>2023-04-24 11:10:00  0        270       0          1          0        0        0.37
<br/>2023-04-24 11:20:00  0        335       0          0          0        0        0.0
<br/>2023-04-24 11:30:00  1        269       0          0          0        0        0.0
<br/>2023-04-24 11:40:00  0        255       0          0          0        0        0.0
<br/>2023-04-24 11:50:00  0        305       0          1          0        0        0.33
<br/>2023-04-24 12:00:00  191      255       132        0          126      0        0.0
<br/>2023-04-24 12:10:00  0        225       0          0          0        0        0.0
<br/>2023-04-24 12:20:00  88       249       84         0          47       0        0.0
<br/>2023-04-24 12:30:00  87       242       0          1          65       0        0.41
<br/>2023-04-24 12:40:00  1        234       0          0          0        0        0.0
<br/>2023-04-24 12:50:00  85       235       74         0          58       0        0.0
<br/>2023-04-24 13:00:00  52       277       29         0          40       0        0.0
<br/>2023-04-24 13:10:00  0        256       0          0          0        0        0.0
<br/>2023-04-24 13:20:00  0        302       0          1          0        0        0.33
<br/>2023-04-24 13:30:00  218      296       102        0          125      0        0.0
<br/>2023-04-24 13:40:00  0        272       0          2          0        0        0.74
<br/>2023-04-24 13:50:00  2        311       0          0          1        0        0.0
<br/>2023-04-24 14:00:00  448      289       64         0          309      0        0.0
<br/>2023-04-24 14:10:00  51       286       24         0          32       0        0.0
<br/>2023-04-24 14:20:00  86       236       0          0          72       0        0.0
<br/>2023-04-24 14:30:00  414      263       44         1          279      0        0.38
<br/>2023-04-24 14:40:00  0        220       0          0          0        0        0.0
<br/>
<br/>======= Slow Queries =======
<br/>query                                                                                                                                                                       nbr_querys  avg_milli  pct_gt120  DC3_querys
<br/>--------------------------------------------------------------------------------------------------------------------------------------------------------------------------  ----------  ---------  ---------  ----------
<br/>SELECT * FROM system.peers                                                                                                                                                  230         119.5      35         171
<br/>SELECT * FROM system.local WHERE key='local'                                                                                                                                219         116.0      37         155
<br/>SELECT keyspace_name, table_name, start_key, end_key, replica_addresses FROM system.partitions                                                                              1277        98.2       30         830
<br/>select customerid,emails,individual407typecode,rescountrycode,customertypecode,inccountrycode,emailremindertimestamp,isemailunsolicited from customer where customerid = ?  783         44.4       1          743
</code>
