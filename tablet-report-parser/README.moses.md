<a name="readme-top"></a>

<!-- PROJECT SHIELDS -->

<h1 align="center">Moses.pl</h1>

<div align="center">
  <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser">
    <img src="moses-dropping-a-tablet.png" alt="Logo" >
  </a>

  <h3 align="center">
    Fetches and analyzes tablets for a Universe</h3>
    <p/>
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser"><strong>Explore TRP docs »</strong></a>
    ·
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser/issues">Report Bug</a>
    ·
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser/issues">Request Feature</a>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
        <li><a href="#Program options">Program options</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#contact">Contact</a></li>

  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

Moses.pl is intended to replace the current (multi-step) process of obtaining and analyzing a tablet report (yugatool/tablet-report-parser).

In a single step, **moses.pl** collects tablet, configuration, xCluster and some metrics information, analyzes it, and produces a 
sqlite database, and a summary report.

In "Wait for Index backfill" mode (--WAIT_INDEX_BACKFILL), moses.pl will wait untill backfill completes. This can be used to automate index creation.
<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- GETTING STARTED -->
## Getting Started

To get a local copy up and running follow these simple example steps:

<a href="https://github.com/yugabyte/yb-tools/blob/main/tablet-report-parser/moses.pl">Download</a> and install the code on any linux host that has access to the YBA host.

You will need the YBA URL, the YBA access token, and the name of the uiverse whose tablets you want to analyze.

### Prerequisites

The host must have perl >= 5.16 installed.

### Installation

<a href="https://github.com/yugabyte/yb-tools/blob/main/tablet-report-parser/moses.pl">Download</a> the code to a suitable directory, and make it executable.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Program options
| Option name  | Value/explanation |
| ------------- |-------------|
|  *--YBA_HOST*        | [=] \<YBA hostname or IP> (Required) "[http(s)://]\<hostname-or-ip>[:\<port>]"|
|  *--API_TOKEN*       | [=] \<API access token>   (Required)|
|  *--UNIVERSE*        | [=] \<Universe-Name-or-uuid>  (Required. Name Can be partial, sufficient to be Unique)|
|  *--CUSTOMER*        | [=] \<Customer-uuid-or-name> (Optional. Required if more than one customer exists)|
|  *--GZIP*            | Use this if you want to create a sql.gz for export, instead of a sqlite DB<br/> In addition, this collects additional debug info as a comment in the SQL.|
|  *--DEBUG*           | Shows additional debugging  information|

   **ADVANCED options**
| Option name  | Value/explanation |
| ------------- |-------------|
|   `--HTTPCONNECT`            |[=] [curl \| tiny]    (Optional. Whether to use 'curl' or HTTP::Tiny(Default))|
|   `--FOLLOWER_LAG_MINIMUM`   |[=] \<value> (milisec)(collect tablet follower lag for values >= this value(default 1000))|
|   `--CONFIG_FILE_(PATH\|NAME)`|[=] <path-or-name-of-file-containing-options> (i.e --CONFIG_FILE_PATH & .._NAME)|
   
   **Backfill related options**
| Option name  | Value/explanation |
| ------------- |-------------|
|   `--WAIT_INDEX_BACKFILL` |        If specified, this program runs till backfills complete. No report or DB.|
|   `--INDEX_NAME         ` |    [=] <idx-name> Optionally Used with WAIT_INDEX_BACKFILL, to specify WHICH idx to wait for.|
|   `--SLEEP_INTERVAL_SEC ` |    [=] nn  Number of seconds to sleep between check for backfill; default 30.|


* If STDOUT is redirected, it can be sent to  a SQL file, or gzipped, and collected for offline analysis.\
   (Similar to `--gzip`)
* You may abbreviate option names up to the minimum required for uniqueness.
* Options can be set via `--cmd-line`, or via environment, or both, or via a "`config_file`".
* We look for config files by default at `--CONFIG_FILE_PATH`=/home/yugabyte with a name "`*.yba.rc`".
* Expected config file content format is : `EXPORT \<OPTION-NAME>="VALUE"`

<!-- USAGE EXAMPLES -->
## Usage
 
 `perl moses.pl  --YBA_HOST=https://Your-yba-hostname-or-IP --API_TOKEN=Your-token  --univ Your-universe-name`

### Sample run
```
$ perl  ./moses.pl --yba=https://yba-hostname --api=f7cd9197-21be-4718-9e6c-xxxxxxx9 --univ=Univ-name
-- 2023-11-27 17:12 -08:00 : Moses version 0.25 @va-win-VBG4Q starting ...
-- 17:12:10 UNIVERSE: Univ-name on gcp ver 2.18.3.0-b75
-- 17:12:12 Processing tablets on Univ-name-n5 5dc185470c564dc39dbd3672efcdfcd3 (10.231.0.84,Idx 5)...
-- 17:12:18 Processing tablets on Univ-name-n1 92b2779d3a5f496fb0ad7b846f1270e4 (10.231.0.66,Idx 1)... (Idx 5 had 1585 tablets, 790 leaders)
-- 17:12:25 Processing tablets on Univ-name-n4 e5a6e1ab30d0498ea457696ef8cf7dbf (10.231.0.192,Idx 4)... (Idx 1 had 1585 tablets, 791 leaders)
-- 17:12:30 Completed Node Processing. (Idx 4 had 1585 tablets, 4 leaders)
-- The following reports are available --
LOG                           namespaces
NODE                          table_sizes
TABLET                        tablecol
UNSAFE_Leader_create          tables
delete_leaderless_be_careful  tablet_replica_detail
ent_tablets                   tablet_replica_summary
gflags                        tablets_per_node
keyspaces                     version_info
leaderless                    xcTable
metrics                       xcluster
namespace_sizes
-- S u m m a r y--
4 Nodes;  4755 Tablets (0 Leaderless). 809 metrics.
1585 tablets have 3 replicas.
-- 17:12:31 COMPLETED. '2023-11-27.va-win-VBG4Q.tabletInfo.Univ.sqlite' Created
         RUN: sqlite3 -header -column 2023-11-27.va-win-VBG4Q.tabletInfo.Univ.sqlite
$
```
<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

<a href="https://github.com/na6vj">NA6VJ</a>

Project Link: [https://github.com/na6vj/yb-tools](https://github.com/na6vj/yb-tools)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

