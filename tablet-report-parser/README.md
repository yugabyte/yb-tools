# Tablet Report Parser

* Reads a tablet report that was created by **yugatool**
* Can process **dump-entities** output
* Can process **tablet_info** created by yugatool
* outputs a *sqlite* database (or SQL stream) that contains parsed data + analysis info
* Can produce various reports, such as leaderless tablets, tablet-count recommendations...
* Can be used to analyze table/tablet status, check replication factors etc.

## See KB article for details :
### https://yugabyte.zendesk.com/knowledge/articles/12124512476045/en-us

_NOTE: The KB article is more up-to-date than this README._

## HOW TO  Run THIS script  - and feed the generated SQL into sqlite3:

   `$ perl tablet_report_parser.pl  tablet-report-09-20T14.out`

## Sample Run:

```
$ ./tablet-report-parser.pl tablet-report.out
Sun Jan 22 00:00:50 2023 ./tablet-report-parser.pl version 0.33
        Reading tablet-report.out,
        creating/updating sqlite db tablet-report.out.sqlite.
./tablet-report-parser.pl Version 0.26 generating SQL on Sun Jan 22 00:00:50 2023
... 1 CLUSTER items processed.
Processing MASTER from [ Masters ] (Line#4)
... 3 MASTER items processed.
Processing TSERVER from [ Tablet Servers ] (Line#10)
... 12 TSERVER items processed.
... ... ...
  ... 871 TABLET items processed.
Main SQL loading Completed. Generating table stats...
--- Completed. Available REPORT-NAMEs ---
cluster                       tableinfo
delete_leaderless_be_careful  tablet
large_tables                  tablet_replica_detail
large_wal                     tablet_replica_summary
leaderless                    tablets_per_node
region_zone_distribution      unbalanced_tables
summary_report                version_info
table_detail
|--- Summary Report ---
     |
     |12 TSERVERs, 129 Tables, 9387 Tablets loaded.
     |45 tablets have RF < 3. (See "tablet_replica_summary/detail")
     |63 tables have unbalanced tablet sizes (see "unbalanced_tables")
     |90 leaderless tablets found.(See "leaderless")
 --- To get a report, run: ---
  sqlite3 -header -column tablet-report-parser/tablet-report.out.sqlite "SELECT * from <REPORT-NAME>"
```

## HOW TO Run Analysis using SQLITE ---
### Identify Leaderless tablets
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
    67da88ffc8a54c63821fa85d82aaf463  table-name-1  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.xxx.yy.zz1
    31a7580dba224fcfb2cc57ec07aa056b  table-name-2  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.xxx.yy.zz1
    9795475a798d411cb25c1627df13a122  table-name-3  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.xxx.yy.zz1
    sqlite>
```
### Estimate tablet counts for large tables
This is useful when the number of tablets gets very large, and impacts system performance.
In this case, the following analysis can help determine the "proper" number of tablets for large tables.
The "proper" number depends on how many nodes are in the cluster - so the report provides values for a few pre-configured node counts.

```
sqlite> select * from large_tables;
NAMESPACE  TABLENAME          uniq_tablets  sst_table_mb  tablet_size_mb  rec_8node_tablets  rec_12node_tablets  rec_24node_tablets  repl_factor  wal_table_mb
---------  -----------------  ------------  ------------  --------------  -----------------  ------------------  ------------------  -----------  ------------
tbl_spc1   table----name---1  96            42997         447             2.1                1.6                 1.0                 3            1152
tbl_spc1   table----name---2  96            20186         210             1.3                1.0                 0.8                 3            9216
tbl_spc1   table----name---3  24            16142         672             1.1                0.9                 0.7                 3            3072
tbl_spc1   table----name---4  96            15757         164             1.1                0.9                 0.7                 3            576
tbl_spc1   table----name---5  24            13097         545             1.0                0.8                 0.7                 3            3072
tbl_spc1   table----name---6  16            8775          548             0.8                0.7                 0.6                 3            32
tbl_spc1   table----name---7  24            6553          273             0.7                0.7                 0.6                 3            3072
tbl_spc1   table----name---8  24            6390          266             0.7                0.7                 0.6                 3            48
tbl_spc1   table----name---9  24            5979          249             0.7                0.6                 0.6                 3            48
tbl_spc1   table----name---x  24            5214          217             0.7                0.6                 0.6                 3            48
sqlite>
```

In this example, the recommendation for "**table----name---1**" is that on a 12-node cluster, it should be configured to have **1.6** tablets, which should be rounded-up to 2 tablets.
The system should be configured to default to creating **1** tablet per table, so that tables NOT identified by this report default to a single tablet.

In case it becomes necessary to get recommendations for a 3-node system, here is the query:

```
SELECT tablename,uniq_tablet_count as uniq_tablets,
      (sst_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as sst_table_mb,
          (sst_tot_bytes /tot_tablet_count/1024/1024) as tablet_size_mb,
          round((sst_tot_bytes /1024.0/1024.0/3.0  + 5000) / 10000,1) as rec_3node_tablets,
           tot_tablet_count / uniq_tablet_count as repl_factor,
      (wal_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as wal_table_mb
        FROM tableinfo
        WHERE sst_table_mb > 5000
        ORDER by sst_table_mb desc;
```

### "Advanced" Run options:
By default, the script will automatically pipe the  output stream to sqlite3, and create a sqlite database named <input-file>.sqlite.
You can change the default behaviour by explicitly piping the output:

$ ./tablet-report-parser.pl tablet-report.out | sqlite3 i.can.name.my.sqlite.db.here.sqlite

Alternatively, you can save the SQL stream to a file, then run sqlite separately:

$ ./tablet-report-parser.pl tablet-report.out > saved.SQL.statements.sql

You can pass multiple file names into the program, the tablet report, entities , or tablet_info:

`perl  tablet-report-parser.pl tablet-report-2023-02-14.out.gz  007347.dump-entities bb3a15e4023a493cba91fdfbd4316570.txt`

 * Files with "entities" in the name are processed as "dump-entities" files. These are created by:
        curl <master-leader-hostname>:7000/dump-entities | gzip -c > 1000 4 24 27 30 46 119 1000 1001date -I)-<master-leader-hostname>.dump-entities.gz

 * Files named "<tablet-uuid>.txt"  are assumed to be "tablet-info" files. These are created by:
       ./yugatool -m $MASTERS $TLS_CONFIG tablet_info $TABLET_UUID > $TABLET_UUID.txt

