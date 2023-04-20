# querymonitor

## Synopsis
Runs (as a daemon) and periodically collects live query info into csv (.gz) files.
Personably Identifyable information (PII) is removed from each query by truncating the WHERE clause.

These are suitable for offline analysis to obtain:
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

By default, the program runs in the background, collecting query samples every 5 seconds,  for 4 hours, then exits.

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
<br/>Program Options:
<br/>        --API_TOKEN
<br/>        --CURL
<br/>        --CUST_UUID
<br/>        --DAEMON
<br/>        --DEBUG
<br/>        --FLAGFILE
<br/>        --HELP
<br/>        --INTERVAL_SEC
<br/>        --RUN_FOR
<br/>        --UNIV_UUID
<br/>        --YBA_HOST
<br/>        --YCQL_OUTPUT
<br/>        --YSQL_OUTPUT
</code>
