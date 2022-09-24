# Tablet Report Parser

* Reads a tablet report that was created by yugatool
* outputs a SQL stream that contains parsed data + analysis info

This can be used to analyze table/tablet status, check replication factors, look for and correct leaderless tablets.

### HOW TO  Run THIS script  - and feed the generated SQL into sqlite3:

   $ perl tablet_report_parser.pl < tablet-report-09-20T14.out | sqlite3 tablet-report-09-20T14.sqlite

### HOW TO Run Analysis using SQLITE ---

 $ sqlite3 -header -column tablet-report-09-20T14.sqlite
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
