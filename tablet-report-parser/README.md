# Tablet Report Parser

* Reads a tablet report that was created by yugatool
* outputs a SQL stream that contains parsed data + analysis info

This can be used to analyze table/tablet status, check replication factors, look for and correct leaderless tablets.

### HOW TO  Run THIS script  - and feed the generated SQL into sqlite3:

   $ perl tablet_report_parser.pl  tablet-report-09-20T14.out | sqlite3 tablet-report-09-20T14.sqlite

### Sample Run:


```
$ ./tablet_report_parser.pl tablet-report-cup2-006430.out | sqlite3 tablet-analysis.sqlite

./tablet_report_parser.pl Version 0.06 generating SQL on Sat Sep 24 12:59:50 2022
... 1 CLUSTER items processed.
Processing line# 4 :MASTER from [ Masters ]
... 3 MASTER items processed.
Processing line# 10 :TSERVER from [ Tablet Servers ]
... 12 TSERVER items processed.
Processing line# 25 :TABLET from [ Tablet Report: [host:"10.184.7.181" port:9100] (bba8024cfc72437baa81ef02a7df3609) ]
... 1 TABLET items processed.
Processing line# 29 :TABLET from [ Tablet Report: [host:"10.185.8.19" port:9100] (3114e1c74509442dbd161577e5a654be) ]
... 2642 TABLET items processed.
...
SQL loading Completed.
--- Available REPORT-NAMEs ---
cluster                 tablet                  tablets_per_table
hexval                  tablet_replica_detail   version_info
leaderless              tablet_replica_summary
summary_report          tablets_per_node
--- Summary Report ---
     |12 TSERVERs, 115 Tables, 31384 Tablets loaded.
     |2322 leaderless tablets found.(See "leaderless")
     |4964 tablets have RF < 3. (See "tablet_replica_summary/detail")
--- To get a report, run: ---
        sqlite3 -header -column tablet-analysis.sqlite "SELECT * from REPORT-NAME"
```

### HOW TO Run Analysis using SQLITE ---

 $ sqlite3 -header -column tablet-analysis.sqlite

```
SQLite version 3.31.1 2020-01-27 19:55:54
    Enter ".help" for usage hints.
    sqlite> select count(*) from leaderless;
    count(*)
    ----------
    2321
    sqlite> select *  from leaderless limit 3;
    tablet_uuid                       table_name    node_uuid                         status                        ip
    --------------------------------  ------------  --------------------------------  ----------------------------  -----------
    67da88ffc8a54c63821fa85d82aaf463  custaccessid  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
    31a7580dba224fcfb2cc57ec07aa056b  packetdocume  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
    9795475a798d411cb25c1627df13a122  packet        5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
    sqlite>
```
