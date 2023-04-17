# querymonitor

## Synopsis
Runs (as a daemon) and periodically collects live query info into csv (.gz) files.

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
--API_TOKEN= \<value><br>
--YBA_HOST = \<value><br>
--CUST_UUID= \<value><BR>
--UNIV_UUID= \<value>
</code>

You can specify the flagfile to use:

<code>
  ./querymonitor.pl --flagfile=myuniv_queryparms.flag
</code>

A <code>--help</code> parameter provides a full list of available flags.