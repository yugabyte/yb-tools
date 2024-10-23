#!/usr/bin/perl
##########################################################################
## Tablet Report Parser
##    Parses the "tablet report" created by yugatool, and creates a sqlite DB.
##    It can also parse "dump-entities" output, and "tablet-info" output.
## See KB: https://yugabyte.zendesk.com/knowledge/articles/12124512476045/en-us

# Run instructions:

# 1: obtain a tablet report. To get that, run Yugatool;:
# (See https://support.yugabyte.com/hc/en-us/articles/6111093115405-How-to-use-the-Yugatool-utility)
#
#		export MASTERS=10.183.11.69:7100,10.184.7.181:7100,10.185.8.17:7100
#		export TLS_CONFIG="--cacert /opt/yugabyte/certs/MyCA.crt"
#		export TLS_CONFIG="$TLS_CONFIG --skiphostverification"
#
#		./yugatool cluster_info \
#		  -m $MASTERS \
#		  $TLS_CONFIG \
#		  --tablet-report \
#		  | gzip -c > /tmp/tablet-report-$(hostname).out.gz
#
#       You can also use the "-o json" option. This code accepts that output also.

# 2: Run THIS script  - which reads the tablet report and generates SQL,
#    and feeds the generated SQL into sqlite3:

#        $ perl tablet_report_parser.pl tablet-report.out[.gz]

#
# EXTRAS:
#   * Input files can be gzip compressed (must end with .gz)
#   * Versions >= 0.32 can parse multiple files (of different types):
#   * Files with "entities" in the name are processed as "dump-entities" files.
#      The "Entities" file is created using:
#          curl <master-leader-hostname>:7000/dump-entities | gzip -c > $(date -I)-<master-leader-hostname>.dump-entities.gz
#   * Files named "<tablet-uuid>.txt"  are assumed to be "tablet-info" files. These are created by:
#         ./yugatool -m $MASTERS $TLS_CONFIG tablet_info $TABLET_UUID > $TABLET_UUID.txt 
##########################################################################
our $VERSION = "0.44";
use strict;
use warnings;
#use JSON qw( ); # Older systems may not have JSON, invoke later, if required.
use MIME::Base64;

BEGIN{ # Namespace forward declarations
  package TableInfo;
  package JSON_Analyzer;
  package entities_parser;
  package Tablet_Info; #Handles  Output from yugatool tablet_info 
} # End of namespace declaration 
my %opt=(
	STARTTIME	=>  unixtime_to_printable(time(),"YYYY-MM-DD HH:MM"),
	DEBUG		=> 0,
	HOSTNAME    => $ENV{HOST} || $ENV{HOSTNAME} || $ENV{NAME} || qx|hostname|,
	JSON        => 0, # Auto set to 1 when  "JSON" discovered. default is Reading "table" style.
	AUTORUN_SQLITE => -t STDOUT , # If STDOUT is NOT redirected, we automatically run sqlite3
	SQLITE_ERROR   => (qx|sqlite3 -version|=~m/([^\s]+)/  ?  0 : "Could not run SQLITE3: $!"), # Checks if sqlite3 can run
	PROCESSED_TYPES=> {TABLET_REPORT => 0, ENTITIES => 0, TABLET_INFO => 0}, # Stats 
);
my %ANSICOLOR = (
	ENCLOSE        => sub{"\e[$_[1]m$_[0]\e[0m"},
	NORMAL         => "\e[0m",
	BOLD           => "\e[1m",
	DARK           => "\e[2m",
	FAINT          => "\e[2m",
	UNDERLINE      => "\e[4m",
	UNDERSCORE     => "\e[4m",
	BLINK          => "\e[5m",
	REVERSE        => "\e[7m",
    CONCEALED      => "\e[8m",	
	BLACK          => "\e[30m",
	RED            => "\e[31m",
	GREEN          => "\e[32m",
	YELLOW         => "\e[33m",
	BLUE           => "\e[34m",
	MAGENTA        => "\e[35m",
	CYAN           => "\e[36m",
	WHITE          => "\e[37m",
	BRIGHT_BLACK   => "\e[90m",
	BRIGHT_RED     => "\e[91m",
	BRIGHT_GREEN   => "\e[92m",
	BRIGHT_YELLOW  => "\e[93m",
	BRIGHT_BLUE    => "\e[94m",
	BRIGHT_MAGENTA => "\e[95m",
	BRIGHT_CYAN    => "\e[96m",
	BRIGHT_WHITE   => "\e[97m",
);
our $USAGE = << "__USAGE__";
  $ANSICOLOR{REVERSE}Tablet Report Parser$ANSICOLOR{YELLOW} $VERSION$ANSICOLOR{NORMAL}
  $ANSICOLOR{GREEN}=========================$ANSICOLOR{NORMAL}
  ## See KB: $ANSICOLOR{UNDERLINE}https://yugabyte.zendesk.com/knowledge/articles/12124512476045/en-us$ANSICOLOR{NORMAL}
  
  The input to this program is a "$ANSICOLOR{CYAN}tablet report$ANSICOLOR{NORMAL}" created by yugatool.
  The default output is a sqlite database containing various reports.

  $ANSICOLOR{REVERSE}TYPICAL/DEFAULT/PREFERRED usage:$ANSICOLOR{NORMAL} 
    
   $ANSICOLOR{BRIGHT_YELLOW} perl $0 $ANSICOLOR{BRIGHT_CYAN}\[TABLET-REPORT-FROM-YUGATOOL] \[ENTITY-file] \[tablet-info-file]... $ANSICOLOR{NORMAL}

  This will create and output database file named TABLET-REPORT-FROM-YUGATOOL.sqlite

  $ANSICOLOR{FAINT}ADVANCED usage1: perl $0 TABLET-REPORT-FROM-YUGATOOL | sqlite3 OUTPUT-DB-FILE-NAME$ANSICOLOR{NORMAL}
  $ANSICOLOR{FAINT}ADVANCED usage2: perl $0 [<] TABLET-REPORT-FROM-YUGATOOL > OUTPUT.SQL$ANSICOLOR{NORMAL}

 $ANSICOLOR{CYAN}* Input files can be gzip compressed (must end with .gz)
 $ANSICOLOR{CYAN}* Files with "entities" in the name are processed as "dump-entities" files. These are created by:
 $ANSICOLOR{BLUE}       curl <master-leader-hostname>:7000/dump-entities | gzip -c > $(date -I)-<master-leader-hostname>.dump-entities.gz
 $ANSICOLOR{CYAN}* Files named "<tablet-uuid>.txt"  are assumed to be "tablet-info" files. These are created by:
 $ANSICOLOR{BLUE}      ./yugatool -m \$MASTERS \$TLS_CONFIG tablet_info \$TABLET_UUID > \$TABLET_UUID.txt $ANSICOLOR{NORMAL}
__USAGE__

if (-t STDIN and not @ARGV){
	# No args supplied, and STDIN is a TERMINAL - show usage and quit.
	print  $USAGE,"\n\n";
	die "ERROR: Input file-name not specified.";
}
my ($SQL_OUTPUT_FH, $output_sqlite_dbfilename); # Output file handle to feed to SQLITE
my @more_input_specified = @ARGV;
@ARGV=(); # Zap it -we will specify each file to feed into <>
if ($more_input_specified[0] =~/\-+he?l?p?/i){
   print  $USAGE,"\n\n";
   exit 1;
}

while (my $inputfilename = shift @more_input_specified){
	# User has specified and argument (Default usage) - we will process it as a filename
	print "SELECT 'Processing $inputfilename....' as info;\n"; 
	-f $inputfilename or die "ERROR: No file '$inputfilename'. try --help";
	if ($inputfilename =~/\.gz$/){# Input is compressed
		print "SELECT '$ANSICOLOR{GREEN} $inputfilename appears to be a compressed file.",
		         "$ANSICOLOR{BRIGHT_GREEN} Auto-gunzipping it on the fly...$ANSICOLOR{NORMAL}' as INFO;\n";
		open (STDIN,"-|", "gunzip -c $inputfilename") or die "ERROR: Could not gunzip $inputfilename: $!";
		$inputfilename = substr($inputfilename,0,-3); # Drop the ".gz"
	}else{
		open (STDIN,"<",  $inputfilename) or die "ERROR: Could not open $inputfilename: $!";
	}

	if (not $output_sqlite_dbfilename){
		Setup_Output_Processing($inputfilename );
    }
	if ($inputfilename =~/entities/i){
		print "SELECT '   ... processing as an ENTITIES file..' as info;\n"; 
		my $entities = entities_parser::->new();
		$entities->Ingest_decode_and_Generate(); # Read from stdin
		$opt{PROCESSED_TYPES}{ENTITIES} ++;
	}elsif ($inputfilename =~/\w{32}\.\w{3,4}$/){
		print "SELECT '   ... processing as an TABLET INFO file..' as info;\n"; 
		my $tabletinfo = Tablet_Info::->new();
        $opt{PROCESSED_TYPES}{TABLET_INFO}++;
	}else{
		print "SELECT '   ... processing as a TABLET REPORT input ..' as info;\n"; 
		Process_tablet_report();
		$opt{PROCESSED_TYPES}{TABLET_REPORT}++;
	}
	#close STDIN; # Causes errors if closed. Works fine leaving parent STDIN open.
}

sub Process_tablet_report{
# Sqlite 3.7 does not support ".print", so we use wierd SELECT statements to print  messages.
print  << "__SQL__";
SELECT '$0 Version $VERSION generating SQL on $opt{STARTTIME}';
CREATE TABLE cluster(type, uuid TEXT PRIMARY KEY, ip, port, region, zone ,role, uptime);
CREATE TABLE tablet (node_uuid,tablet_uuid TEXT , table_name,table_uuid, namespace,state,status,
                  start_key, end_key, sst_size INTEGER, wal_size INTEGER, cterm, cidx, leader, lease_status);
CREATE UNIQUE INDEX tablet_idx ON tablet (node_uuid,tablet_uuid);

CREATE VIEW table_detail AS
     SELECT  namespace,table_name, count(*) as total_tablet_count,count(DISTINCT tablet_uuid) as unique_tablet_count, count(DISTINCT node_uuid) as nodes
	 FROM tablet GROUP BY namespace,table_name;
CREATE VIEW tablets_per_node AS
    SELECT node_uuid,ip as node_ip,zone,  count(*) as tablet_count,
           sum(CASE WHEN status='TABLET_DATA_COPYING' THEN 1 ELSE 0 END) as copying,
           sum(CASE WHEN tablet.state = 'TABLET_DATA_TOMBSTONED' THEN 1 ELSE 0 END) as tombstoned,
           sum(CASE WHEN node_uuid = leader THEN 1 ELSE 0 END) as leaders,
           count(DISTINCT table_name) as table_count
        FROM tablet,cluster
        WHERE cluster.type='TSERVER' and cluster.uuid=node_uuid
        GROUP BY node_uuid, ip, zone 
    UNION
    SELECT '~~TOTAL~~',
	    '*(All '|| (select count(*) from cluster where type='TSERVER') || ' nodes)*', 'ALL',
       (Select count(*) from tablet),(Select count(*) from tablet WHERE status='TABLET_DATA_COPYING'),
	   (SELECT count(*) from tablet where state = 'TABLET_DATA_TOMBSTONED'),
	   (SELECT count(*) from tablet where node_uuid = leader),
	   (SELECT count(DISTINCT table_name) as table_count from tablet)
	   ORDER BY 1;
CREATE VIEW tablet_replica_detail AS
	SELECT t.namespace,t.table_name,t.table_uuid,t.tablet_uuid,
    sum(CASE WHEN t.status = 'TABLET_DATA_TOMBSTONED' THEN 0 ELSE 1 END) as replicas  ,
    sum(CASE WHEN t.status = 'TABLET_DATA_TOMBSTONED' THEN 1 ELSE 0 END) as tombstoned,
	sum(CASE when t.node_uuid = leader AND lease_status='HAS_LEASE'
	    then 1 else 0 END ) as leader_count
	from tablet t
	GROUP BY t.namespace,t.table_name,t.table_uuid,t.tablet_uuid;
CREATE VIEW tablet_replica_summary AS
	SELECT replicas,count(*) as tablet_count FROM  tablet_replica_detail GROUP BY replicas;
CREATE VIEW leaderless AS 
     SELECT t.tablet_uuid, replicas,t.namespace,t.table_name,node_uuid,status,ip ,leader_count
	 from tablet t,cluster ,tablet_replica_detail trd
	 WHERE  cluster.type='TSERVER' AND cluster.uuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid  AND t.status != 'TABLET_DATA_TOMBSTONED'
		   AND trd.leader_count !=1;

CREATE VIEW delete_leaderless_be_careful AS 
     SELECT '\$HOME/tserver/bin/yb-ts-cli delete_tablet '|| tablet_uuid ||' -certs_dir_name \$TLSDIR -server_address '||ip ||':9100  \$REASON_tktnbr'
	   AS generated_delete_command
     FROM leaderless;
	 
--  Based on  yb-ts-cli unsafe_config_change <tablet_id> <peer1> (undocumented)
--  https://phorge.dev.yugabyte.com/D12312
CREATE VIEW UNSAFE_Leader_create AS
    SELECT  '\$HOME/tserver/bin/yb-ts-cli --server_address='|| ip ||':'||port 
        || ' unsafe_config_change ' || t.tablet_uuid
		|| ' ' || node_uuid
		|| ' -certs_dir_name \$TLSDIR;sleep 30;' AS cmd_to_run
	 from tablet t,cluster ,tablet_replica_detail trd
	 WHERE  cluster.type='TSERVER' AND cluster.uuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid  AND t.status != 'TABLET_DATA_TOMBSTONED'
		   AND trd.leader_count !=1
		   AND t.state='RUNNING';
		   
CREATE VIEW large_wal AS 
  SELECT table_name, count(*) as tablets,  
        sum(CASE WHEN wal_size >=128000000 then 1 else 0 END) as "GE128MB",
        sum(CASE WHEN wal_size >=96000000  AND  0+rtrim(wal_size,' MB') < 128000000 then 1 else 0 END) as "GE96MB",
        sum(CASE WHEN wal_size >=65000000  AND  0+rtrim(wal_size,' MB') < 96000000 then 1 else 0 END) as "GE65MB",
        sum(CASE WHEN wal_size < 65000000                                  then 1 else 0 END) as "LT65MB"
  FROM tablet 
  WHERE wal_size like '%MB' 
  GROUP by table_name 
  ORDER by GE128MB desc, GE96MB desc;

CREATE VIEW unbalanced_tables AS 
SELECT t.namespace , t.table_name, total_tablet_count,
      unique_tablet_count,nodes,  
       (SELECT tablet_uuid 
		  FROM tablet x 
		  WHERE x.namespace =t.namespace   and  x.table_name =t.table_name 
		     and (x.sst_size +x.wal_size ) = max_tablet_size
    		 LIMIT 1) as large_tablet, 
      round(min_tablet_size/1024.0/1024.0,1) as min_tablet_mb,
      round(max_tablet_size/1024.0/1024.0,1) as max_tablet_mb
FROM 
  (SELECT  namespace,table_name, count(*) as total_tablet_count,
     count(DISTINCT tablet_uuid) as unique_tablet_count,
     count(DISTINCT node_uuid) as nodes,
     round(max(sst_size + wal_size)/min(sst_size+wal_size+0.1),1) as heat_level,
     max(sst_size + wal_size) as max_tablet_size,
     min(sst_size+wal_size) as min_tablet_size
  FROM tablet t
  GROUP BY namespace,table_name
  HAVING heat_level > 2.5
  ORDER BY heat_level desc) t  ;
 
 CREATE VIEW region_zone_distribution AS
 SELECT namespace,region,zone,
       count(*) as tablets,
       count(DISTINCT tablet_uuid) as uniq_tablet,
       sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
       count(DISTINCT table_name) as tables,
       count(DISTINCT node_uuid) as tservers,
       (SELECT count(*) from cluster c1  WHERE type='MASTER' and c1.zone=c.zone and c1.region=c.region) as masters
FROM tablet,cluster c 
WHERE c.uuid=node_uuid 
GROUP  BY namespace,region,zone
UNION 
SELECT namespace,region,'~'||namespace||' Total~',
       count(*) as tablets,
       count(DISTINCT tablet_uuid) as uniq_tablet,
       sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
       count(DISTINCT table_name) as tables,
       count(DISTINCT node_uuid) as tservers,
       (SELECT count(*) from cluster c1  WHERE type='MASTER' and c1.region=c.region) as masters
FROM tablet,cluster c 
WHERE c.uuid=node_uuid 
GROUP  BY namespace,region
UNION 
SELECT '~Total~','~ALL~','~ALL~',
       count(*) as tablets,
       count(DISTINCT tablet_uuid) as uniq_tablet,
       sum(CASE WHEN leader=c.uuid THEN 1 ELSE 0 END) as leaders,
       count(DISTINCT table_name) as tables,
       count(DISTINCT node_uuid) as tservers,
       (SELECT count(*) from cluster c1  WHERE type='MASTER' ) as masters
FROM tablet,cluster c 
WHERE c.uuid=node_uuid 
ORDER BY namespace,region,zone;

-- table to handle hex values from 0x0000 to 0xffff (Not requird) 
--CREATE table hexval(h text primary key,i integer, covered integer);
--WITH RECURSIVE
--     cnt(x) AS (VALUES(0) UNION ALL SELECT x+1 FROM cnt WHERE x<0xffff)
--    INSERT INTO hexval  SELECT printf('0x%0.4x',x) ,x, NULL  FROM cnt;

-- VIEW: unbalanced_tables_tablet_count_per_size
CREATE VIEW unbalanced_tables_tablet_count_per_size AS
SELECT
    table_name,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 < 2048 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 <2048 THEN 1 ELSE 0 END) END AS LT2GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) END AS s2GB_3GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) END AS s3GB_4GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) END AS s4GB_6GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) END AS s6GB_8GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) END AS s8GB_10GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) END AS s10GB_12GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) END AS s12GB_14GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) END AS s14GB_16GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) END AS s16GB_20GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) END AS s20GB_24GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) END AS s24GB_28GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) END AS s28GB_32GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) END AS s32GB_36GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) END AS s36GB_40GB,
    CASE WHEN SUM(CASE WHEN sst_size/1024/1024 > 40960 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN sst_size/1024/1024 > 40960 THEN 1 ELSE 0 END) END AS GT40GB
FROM
    tablet
WHERE
    lease_status = 'HAS_LEASE'
    AND table_name IN (SELECT table_name FROM unbalanced_tables)
GROUP BY
    table_name
ORDER BY
    2 DESC;

CREATE VIEW version_info AS 
    SELECT '$0' as program, '$VERSION' as version, '$opt{STARTTIME}' AS run_on, '$opt{HOSTNAME}' as host;
__SQL__
# summary_report is moved towards the end, for older SQLITE which requires ALL components to exist when view created.

my ( $line, $json_line, $current_entity);
my %entity = (
    REPORTINFO => {REGEX => '^\[ ReportInfo \]', HANDLER=>sub{print "--$.:$line\n"}, COUNT=>0},
    CLUSTER => {REGEX=>'^\[ Cluster \]', HANDLER=>\&Parse_Cluster_line,              COUNT=>0},
	MASTER  => {REGEX=>'^\[ Masters \]',       		 HANDLER=>\&Parse_Master_line},
	TSERVER => {REGEX=>'^\[ Tablet Servers \]',		 HANDLER=>\&Parse_Tserver_line },
	TABLET  => {REGEX=>'^\[ Tablet Report: ', 
	            HDR_EXTRACT => sub{$_[0] =~m/\[host:"([^"]+)"\s+port:(\d+)(.*?)\] \((\w+)/},
				HDR_KEYS    => [qw|HOST PORT EXTRA_INFO NODE_UUID|], # EXTRA_INFO could have another IP/PORT (YB-managed)
				HANDLER=>\&Parse_Tablet_line,
				LINE_REGEX =>
                 	qr| ^\s(?<tablet_uuid>(\w{32}))\s{3}
					(?<tablename>([\w\-.]+))\s+
					(?<table_uuid>(\w{32})?)(:?\.colocation\.parent\.uuid)?\s* # This exists only if --show_table_uuid is set
					(?<namespace>([\w\-]+))\s+
					(?<state>(\w+))\s*
					(?<status>(\w+))\s+
					(?<start_key>(0x\w+)?)\s* # This could be EMPTY. If so, start_key value will contain END!
					(?<end_key>(0x\w+)?)\s*   # This could also be EMPTY, but start exists in that case
					(?<sst_size>(\-?\d+))  \s  (?<sst_unit>(\w+)) \s+  # "0 bytes"/"485 kB"/"12 MB"
					(?<wal_size>(\-?\d+))  \s  (?<wal_unit>(\w+)) \s+
					(?<cterm>([\[\]\d]+))\s*
					(?<cidx>(-?[\[\]\d]+))\s+       #  Can have a leading minus 
					(?<leader>([\[\]\w]+))\s+
					(?<lease_status>([\[\]\w]+)?)|x,
									 
	},
);

my $entity_regex = join "|", map {$entity{$_}{REGEX}} keys %entity;

$current_entity = "CLUSTER"; # This line should have been in the report, but first item is the cluster 
#--- Main input & processing loop -----
while ($line=<>){
	if ($opt{JSON}){
		$json_line .= $line;
	    if ($line =~m/^\s?}[\s,]*$/){
		   # Process previous JSON segment
		   Set_Transaction(1);
		   JSON_Analyzer::Process_line($json_line);
		   Set_Transaction(0);
		   #$tot_bytes += length($json_line);
		   $json_line = ""; # Zap it   
	    }
		next;
	}
	if (length($line) < 5){ # Blank line ?
	   if ($. < 2  and  $line=~/^\s*{\s*$/ ){
		   $opt{JSON}=1;
		   print "SELECT '-- JSON input detected. ---';\n";
		   $json_line .= $line;
	   }
	   next;
	}

	if ($line =~m/$entity_regex/){
		print "SELECT '... $entity{$current_entity}{COUNT} $current_entity items processed.';\n"; # For previous enity 
	    ($current_entity) = grep {$line =~m/$entity{$_}{REGEX}/ } keys %entity;
		chomp $line;
		print "SELECT 'Processing $current_entity from $line (Line#$.)';\n";
		$entity{$current_entity}{COUNT} = 0;
		Set_Transaction(1,"Starting $current_entity");
		next unless my $extract_sub = $entity{$current_entity}{HDR_EXTRACT};
		my @extracted_values = $extract_sub->($line,$current_entity,\%entity);
		$entity{$current_entity}{$_} = shift @extracted_values for @{$entity{$current_entity}{HDR_KEYS}};
		print "--     for ", map({"$_=$entity{$current_entity}{$_} "} @{$entity{$current_entity}{HDR_KEYS}} ),"\n";
		next;
	}
    if (substr($line,0,1) eq " "  and  $line =~/^ [A-Z_\s]+$/ ){
		Process_Headers($line,$current_entity,\%entity);
		print "--Headers:", map({$_->{NAME} . "(",$_->{START},"), "} @{$entity{$current_entity}{HEADERS}}),"\n";
		next;
	}
    die "ERROR:Line $.: No context for $line" unless $current_entity;
	$entity{$current_entity}{HANDLER}->($line,$current_entity,\%entity); # $line is global
	$entity{$current_entity}{COUNT}++;
}
# --- End of Main loop ----
Set_Transaction(0);

##if ($opt{JSON}){
##	# no table report
##}else{
	print << "__MAIN_COMPLETE__";
SELECT '  ... $entity{$current_entity}{COUNT} $current_entity items processed.';
SELECT 'Main SQL loading Completed. Generating table stats...';
__MAIN_COMPLETE__
	Set_Transaction(1);
	TableInfo::Table_Report();
	Set_Transaction(0);
##}
}
#---------------------------------------------------------------------------------------------
my $tmpfile = "/tmp/tablet-report-analysis-settings$$";
my $extra_tablet_reports="";
$opt{PROCESSED_TYPES}{ENTITIES} and $extra_tablet_reports = ",extra_Tablets_summary/detail";

print << "__ENDING_STUFF__";
SELECT '--- Completed. Available REPORT-NAMEs ---';
.tables
--- Summary report ----
CREATE VIEW summary_report AS
	SELECT (SELECT count(*) from cluster where type='TSERVER') || ' TSERVERs, ' 
	     || (SELECT count(DISTINCT table_name) FROM tablet) || ' Tables, ' 
		 || (SELECT count(*) from tablet) || ' Tablets loaded.'
		AS Summary_Report
	UNION 
	 SELECT count(*) || ' leaderless tablets found.(See "leaderless")' FROM leaderless
	UNION
	  SELECT sum(GE128MB) || ' wal files in ' ||sum(CASE WHEN GE128MB>0 then 1 else 0 END) 
	        || ' tables are > 128MB' FROM large_wal
	UNION
	 SELECT (SELECT sum(tablet_count) FROM tablet_replica_summary WHERE replicas < 3) 
			 || ' tablets have RF < 3. (See "tablet_replica_summary/detail")'
	UNION
	   SELECT count(*) || ' tables have unbalanced tablet sizes (see "unbalanced_tables")' 
	   from unbalanced_tables 
	UNION
	   SELECT count(*) || ' Zones have unbalanced tablets (See "region_zone_tablets$extra_tablet_reports")'
	    from  region_zone_tablets WHERE balanced='NO'
	;
SELECT '','--- Summary Report ---'
UNION 
SELECT '     ',* FROM summary_report;
.output $tmpfile
.databases
.output stdout
.quit
__ENDING_STUFF__
close STDOUT; # Done talking to sqlite3
if ($opt{AUTORUN_SQLITE}){
	close $SQL_OUTPUT_FH;
	select STDOUT;
}
open  STDOUT, ">", "/dev/null"; # To avoid Warning about "Filehandle STDOUT reopened as xxx"
my $retry = 30; # Could take 30 sec to generate summary report...
while ($retry-- > 0 and ! -e $tmpfile){
  sleep 1; # Wait for sqlite to close up 
}

# Get filename output by ".databases" above..
open my $tmp, "<", $tmpfile or exit 0; # Ignore if missing 
my $sql_db_file;
while (<$tmp>){
	m/main:?\s+(\S+)/ or next;
	$sql_db_file = $1;
	last;
}
close $tmp;
$sql_db_file or exit 1; # Did not find the name
warn " --- To get a report, run: ---\n",
     qq|  sqlite3 -header -column $sql_db_file "SELECT * from <REPORT-NAME>"\n|;
unlink $tmpfile;
exit 0;
#-------------------------------------------------------------------------------------
sub Setup_Output_Processing{
	my ($inputfilename) = @_;
	$inputfilename=~/\.json$/i and $inputfilename=substr($inputfilename,0,-5); # Drop the ".json" 
	$inputfilename=~/\.out$/i and $inputfilename=substr($inputfilename,0,-4); # Drop the ".out"
	$output_sqlite_dbfilename = $inputfilename . ".sqlite";
	if (not $opt{AUTORUN_SQLITE}){
		return; # No output processing needed-this has been setup manually
	}
	if ($opt{SQLITE_ERROR}){
		print $USAGE;
		die "ERROR: $opt{SQLITE_ERROR}";
	}
	if (-e $output_sqlite_dbfilename){
		my $mtime     = (stat  $output_sqlite_dbfilename)[9];
		my $rename_to = $inputfilename .".". unixtime_to_printable($mtime,"YYYY-MM-DD") . ".sqlite";
		if  (-e $rename_to){
			die "ERROR:Files $output_sqlite_dbfilename and  $rename_to already exist. Please cleanup!";
		} 
		print "WARNING: Renaming Existing file $output_sqlite_dbfilename to $rename_to.\n";
		rename $output_sqlite_dbfilename, $rename_to or die "ERROR:cannot rename: $!";
		sleep 2; # Allow time to read the message 
	}
	print  $opt{STARTTIME},$ANSICOLOR{BRIGHT_BLUE}," $0 version $VERSION\n",
		$ANSICOLOR{BRIGHT_GREEN},"\tReading $inputfilename,\n",$ANSICOLOR{NORMAL},
		"\tcreating/updating sqlite db $ANSICOLOR{BRIGHT_RED}$output_sqlite_dbfilename$ANSICOLOR{NORMAL}.\n";

	open ($SQL_OUTPUT_FH, "|-", "sqlite3 $output_sqlite_dbfilename")
		or die "ERROR: Could not start sqlite3 : $!";
	select $SQL_OUTPUT_FH; # All subsequent "print" goes to this file handle.
}
#-------------------------------------------------------------------------------------
sub Parse_Cluster_line{
	my ($line,$current_entity, $entity) =@_;
	my ($uuid,$zone) = $line=~m/^\s*([\w\-]+).+(\[.*\])/; 
	if (! $zone){ # Zone may not have been enclosed in []
	   my @piece = split /\s+/,	$line;
	   $piece[0] eq '' and shift @piece; # First piece is empty because of leading blanks in $line 
	   $uuid = $piece[0];
	   my ($zone_idx) = grep { $entity->{$current_entity}{HEADERS}[$_]->{NAME} eq "ZONES" } 0..$#{ $entity->{$current_entity}{HEADERS} };
	   $zone = $piece[$zone_idx];
	}
	print "INSERT INTO cluster(type,uuid,zone) VALUES('CLUSTER',",
	       "'$uuid','$zone');\n";
    # "CLUSTER" is the default type of line, and only ONE such line should exist EARLY in the file.
	# Bail out If we find we are processing CLUSTER lines after 9  records		   
    if ($. > 9){
	   print "SELECT 'ERROR: This does not appear to be a TABLET REPORT (too many CLUSTER lines)';\n";	
	   die "ERROR: This does not appear to be a TABLET REPORT (too many 'CLUSTER' lines)";	
	}
}
sub Parse_Master_line{
	my ($line) =@_;	
	my ($uuid, $host, $port, $region,$zone,$role) = $line=~m/(\S+)/g;
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,role)\n",
	      "  VALUES('MASTER','",
          join("','", $uuid,$host,$port,$region,$zone,$role),
		  "');\n";
}

sub Parse_Tserver_line{
	my ($line,$current_entity, $entity) =@_;
	if (substr($line,0,1) eq "{"){ # Some sort of error message - ignore
	    chomp $line;
		print "SELECT 'ERROR in input line# $. : ",substr($line,0,40)," ... ignored.';\n";
		return;
	}
	my ($uuid,$host,$port,$region,$zone,$alive,$reads,$writes,$heartbeat,$uptime,
        $sst_size,$sst_uncompressed,$sst_files,$memory)
		= split /\s\s+/,$line ;
	$uuid =~s/^\s+//g; #Zap leading space
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,uptime)\n",
	      "  VALUES('TSERVER','",
          join("','", $uuid,$host,$port,$region,$zone,$uptime),
		  "');\n";
    $entity->{TSERVER}{BY_UUID}{$uuid}={
		HOST=>$host,PORT=>$port,REGION=>$region,ZONE=>$zone,UPTIME=>$uptime
	        };
	TableInfo::->Register_Region_Zone($region, $zone, $uuid);
}

sub Parse_Tablet_line{
	my ($line,$current_entity, $entity) =@_;
# 0a2aa531ce7541f4bfffc634200d16c5   brokerageaccountphone                                          titan_prod   RUNNING   TABLET_DATA_READY   0x728e      0x7538    21 MB      2048 kB    27      288541     b686d09824b4455997873522dedcd3a9   HAS_LEASE
	my %kilo_multiplier=(
		BYTES	=> 1,
		KB		=> 1024,
		MB		=> 1024*1024,
		GB		=> 1024*1024*1024,
	);
    if ($line =~ $entity->{$current_entity}{LINE_REGEX}){
		# Fall through and process it
	}else{
	    #Regex failed to match 
		print "SELECT 'ERROR: Line $. failed to match tablet regex';\n"; # so sqlite can show the error also 
        die "ERROR: Line $. failed to match tablet regex";		
	}
	my %save_val=%+; # Save collected regex named capture hash (before it gets clobbered by next regex)
    if ($save_val{namespace} eq "RUNNING"  or  $save_val{namespace} eq "NOT_STARTED"){
	   # We have mis-interpreted this line because NAMESPACE wa missing - re-interpret without namespace
       $line =~m/^\s(?<tablet_uuid>(\w{32}))\s{3}
					(?<tablename>([\w\-]+))\s+
					(?<table_uuid>(\w{32})?)\s* # This exists only if --show_table_uuid is set
					# NOTE: <namespace> has been REMOVED from this regex
					(?<state>(\w+))\s+
					(?<status>(\w+))\s+
					(?<start_key>(0x\w+)?)\s* 
					(?<end_key>(0x\w+)?)\s*  
					(?<sst_size>(\-?\d+))  \s  (?<sst_unit>(\w+)) \s+  # "0 bytes"|"485 kB"|"12 MB"
					(?<wal_size>(\-?\d+))  \s  (?<wal_unit>(\w+)) \s+
					(?<cterm>([\[\]\d]+))\s*    # This could be "[]" or a number.. 
					(?<cidx>([\[\]\d\-]+))\s+
					(?<leader>([\[\]\w]+))\s+
					(?<lease_status>([\[\]\w]+)?) 
                  /x or die "ERROR parsing tablet line#$.";					
	    print "-- line#$. : No NAMESPACE found for table '$+{tablename}' tablet $+{tablet_uuid}\n";
	    %save_val=%+; # clobber it with new info
		$save_val{namespace} = '';
	}
	for (qw|cterm cidx leader lease_status|){
		next unless $save_val{$_} eq "[]";
		$save_val{$_}= ''; # Zap it 
	}
	$save_val{sst_size} *= ($kilo_multiplier{ uc $save_val{sst_unit} } || 1);
	$save_val{wal_size} *= ($kilo_multiplier{ uc $save_val{wal_unit} } || 1);
	print "INSERT INTO tablet (node_uuid,tablet_uuid , table_name,table_uuid,namespace,state,status,",
                  "start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status) VALUES('",
				  $entity->{TABLET}{NODE_UUID},"'", 
	              map({ ",'" . ($save_val{$_}||'') . "'" } 
        		  qw|tablet_uuid tablename table_uuid namespace  state status  start_key end_key sst_size  wal_size 
             		  cterm cidx leader lease_status|  
		  ),");\n";

	if ( $save_val{start_key} and !$save_val{end_key}
	    and $line =~/\s{12}$save_val{start_key}   /
	){
	   # Special case - in $line, $start key is missing, but regex mistakenly places end at start.
	   print "UPDATE tablet SET start_key='', end_key='$save_val{start_key}' ",
	         " WHERE tablet_uuid='$save_val{tablet_uuid}' AND node_uuid='",
			 $entity->{TABLET}{NODE_UUID}, "'; -- correction for line $.\n";
       $save_val{end_key}   = $save_val{start_key};
       $save_val{start_key} = undef;	   
	}
	
    TableInfo::find_or_new( \%save_val )
	        ->collect(\%save_val, $entity->{TABLET}{NODE_UUID});
}

sub Process_Headers{
	my ($line,$current_entity, $entity) =@_;
	$entity->{$current_entity}{PREVIOUS_HEADERS} = $entity->{$current_entity}{HEADERS};
	$entity->{$current_entity}{HEADERS}=[]; # Zap it 
	my $hdr_idx = 0;
	while ( $line =~/([A-Z_]+)/g ){
		my $hdr_item = $1;
		
		$entity->{$current_entity}{HEADERS}[$hdr_idx] =  {NAME=>$hdr_item, START=>$-[0], END=> $+[0], LEN=> $+[0] - $-[0]};
		$hdr_idx++;
	}
}

BEGIN{
	 my $in_transaction = 0; # Private/static  var

sub Set_Transaction{
   my ($start, $msg)=@_;	
   if ($start){
      if ($in_transaction){
	      Set_Transaction(0); # End the previous tx
	  }
	  $msg and print "-------- $msg -----\n";
	  print "BEGIN TRANSACTION;\n";
	  $in_transaction = 1;
	  return;
   }
   if ($in_transaction){
	   # Close transaction
	   $msg and print "-------- $msg -----\n";
	   print "COMMIT;\n\n";
	   $in_transaction = 0;
   }else{
	   # no-op
   }
}
}
sub unixtime_to_printable{
	my ($unixtime,$format) = @_;
	my ($sec,$min,$hour,$mday,$mon,$year,$wday,$yday,$isdst) = localtime($unixtime);
	if (not defined $format  or  $format eq "YYYY-MM-DD HH:MM:SS"){
       return sprintf("%04d-%02d-%02d %02d:%02d:%02d", $year+1900, $mon+1, $mday, $hour, $min, $sec);
	}
	if (not defined $format  or  $format eq "YYYY-MM-DD HH:MM"){
       return sprintf("%04d-%02d-%02d %02d:%02d", $year+1900, $mon+1, $mday, $hour, $min);
	}	
	if ($format eq "YYYY-MM-DD"){
		  return sprintf("%04d-%02d-%02d", $year+1900, $mon+1, $mday);
	}
	die "ERROR: Unsupported format:'$format' ";
}
#=============================================================================================
#  C l a s s e s 
#=============================================================================================
####################################################################################
BEGIN{
package TableInfo; # Also provides Region/Zone info

my %collection = (); # Collection of Tableinfo objects, key is namespace:table_name:uuid  
our %region_zone = ();
our %Tserver_to_region_zone=();
our %Tablet_by_region_zone = ();
our %Tablet_bytes=(); # wal + SST, in bytes, indexed by tablet_uuid 
our $tablet_bucket_max     = 0; 
my %field      = (   # Key=Database field name
	TABLE_UUID		=>{TYPE=>'TEXT', SOURCE=>'table_uuid', SEQ=>3},
	NAMESPACE		=>{TYPE=>'TEXT', SOURCE=>'namespace' , SEQ=>1},
	TABLENAME		=>{TYPE=>'TEXT', SOURCE=>'tablename',  SEQ=>2},
	TOT_TABLET_COUNT=>   {TYPE=>'INTEGER',VALUE=>0, SEQ=>4},
	UNIQ_TABLET_COUNT=>  {TYPE=>'INTEGER',VALUE=>0,  INSERT=>sub{return 
	                                    "(SELECT unique_tablet_count from temp_table_detail WHERE table_name='" 
	                                                             . $_[0]->{TABLENAME} . "' and namespace='"
																 . $_[0]->{NAMESPACE} . "')"}
						, SEQ=>5} ,
	## KEYRANGELIST	=>  {TYPE=>'INTEGER',VALUE=>[], INSERT=>sub{return scalar(@{ $_[0]->{KEYRANGELIST} })}, SEQ=>6} ,
	UNIQ_TABLETS_ESTIMATE => {TYPE=>'INTEGER',VALUE=>0,   SEQ=>7},
	LEADER_TABLETS   => {TYPE=>'INTEGER',VALUE=>0,   SEQ=>8},
	NODE_TABLET_MIN  => {TYPE=>'INTEGER',VALUE=>{}, INSERT=>sub{my $n=9e99; $_ < $n? $n=$_:0 for values %{$_[0]->{NODE_TABLET_COUNT}}; $n}, SEQ=>9} ,
	NODE_TABLET_MAX  => {TYPE=>'INTEGER',VALUE=>{}, INSERT=>sub{my $n=0; $_ > $n? $n=$_:0 for values %{$_[0]->{NODE_TABLET_COUNT}}; $n}, SEQ=>10} ,
    KEYS_PER_TABLET	 => {TYPE=>'INTEGER',VALUE=>0, SEQ=>11 },
	KEY_RANGE_OVERLAP=> {TYPE=>'INTEGER',VALUE=>0, SEQ=>12 },
    UNMATCHED_KEY_SIZE=>{TYPE=>'INTEGER',VALUE=>0, SEQ=>13 },	
	COMMENT           =>{TYPE=>'TEXT'   ,VALUE=>'',SEQ=>14 },
	SST_TOT_BYTES     =>{TYPE=>'INTEGER',VALUE=>0, SEQ=>15 },
	WAL_TOT_BYTES     =>{TYPE=>'INTEGER',VALUE=>0, SEQ=>16 },
	SST_TOT_HUMAN     =>{TYPE=>'TEXT',VALUE=>0, SEQ=>17, INSERT=>sub{MetricUnit::format_kilo($_[0]->{SST_TOT_BYTES})} },
	WAL_TOT_HUMAN     =>{TYPE=>'TEXT',VALUE=>0, SEQ=>18, INSERT=>sub{MetricUnit::format_kilo($_[0]->{WAL_TOT_BYTES})} },
	SST_RF1_HUMAN     =>{TYPE=>'TEXT',VALUE=>0, SEQ=>19, INSERT=>sub{MetricUnit::format_kilo(
	                                          $_[0]->{SST_TOT_BYTES}*$_[0]->{UNIQ_TABLETS_ESTIMATE}/$_[0]->{TOT_TABLET_COUNT}
	                                        )} },
	TOT_HUMAN         =>{TYPE=>'TEXT',VALUE=>0, SEQ=>20, INSERT=>sub{MetricUnit::format_kilo($_[0]->{WAL_TOT_BYTES} + $_[0]->{SST_TOT_BYTES})} },	
);

sub find_or_new{
   my ($t)  = @_; # Parsed tablet hashref 
   my $name = $t->{table_uuid} ||  $t->{namespace} . ":" . $t->{tablename};
   $collection{$name} and return $collection{$name};
   
   $collection{$name} = bless { 
                               map {my $s = $field{$_}{SOURCE} ;
							        $s ? ($_ => $t->{$s}) : () }
									keys %field
							   }
                      , __PACKAGE__;
   return $collection{$name};
}

sub collect{
    my ($self, $tablet, $node_uuid) = @_; # Hash ref of tablet field info
	$self->{TOT_TABLET_COUNT}++;
	$self->{NODE_TABLET_COUNT}{$node_uuid}++;
	$self->{SST_TOT_BYTES} += $tablet->{sst_size};
	$self->{WAL_TOT_BYTES} += $tablet->{wal_size};
	for ($tablet->{start_key}, $tablet->{end_key}){ # Sanity check hex values 
	    next if not defined $_; # Empty is ok 
	    next if length($_) <=6; # Reasonable hex values
        $_ = '0x' . substr($_,-4);	# Use last 4 char 	
	}
	my $start_key = hex($tablet->{start_key} || '0x0000'); # Convert to a binary number 
	my $end_key   = hex($tablet->{end_key}   || '0xffff');
	
	if ($end_key < $start_key) {
		print "--- ERROR: Rec#$.:'END key'= $end_key($tablet->{end_key}) is less than 'start key'=$start_key($tablet->{start_key})\n";
		$end_key=$start_key;
	}
	if (0 == ($self->{UNIQ_TABLETS_ESTIMATE}||=0)){
	   # Need to calcuate this 	
	   $self->{KEYS_PER_TABLET}        = $end_key - $start_key; # keys < $end key, so don't add 1. 
	   $self->{UNIQ_TABLETS_ESTIMATE} = int( (0xffff) / ($self->{KEYS_PER_TABLET} || 1) ); # Truncate decimals 
	}
	$tablet->{lease_status} eq 'HAS_LEASE' and $self->{LEADER_TABLETS}++;
	if ($end_key == 0xffff){
		# The tablet containing the LAST key can have different numbers of keys - so ignore this 
	}elsif (($end_key - $start_key) ==  $self->{KEYS_PER_TABLET}){
		# Matches previous tablets .. all is well
	}else{
		#print ".print ERROR:Line $.: Tablet $tablet->{tablet_uuid} offsets $end_key - $start_key dont match diff=$self->{KEYS_PER_TABLET}\n";
		$self->{UNMATCHED_KEY_SIZE}++;
	}
	if ($self->{KEYS_PER_TABLET} == 0){
		$self->{KEY_RANGE_OVERLAP}++
	}else{
		# start_key must be  an integer multiple of KEYS_PER_TABLET. If not, increment OVERLAP.
		$start_key % $self->{KEYS_PER_TABLET} != 0 and $self->{KEY_RANGE_OVERLAP}++; 
        # Keep a list of key ranges - will check for holes  in this later..
		$self->{KEYRANGELIST}[int($start_key / $self->{KEYS_PER_TABLET}) ] ++;
	}
	my ($region,$zone) = @{ $Tserver_to_region_zone{ $node_uuid } };
	return if $tablet->{status} eq 'TABLET_DATA_TOMBSTONED';
    ##$region_zone{$region}{$zone}{TABLET}{ $tablet->{tablet_uuid} }++;
	my $replicas = ++$Tablet_by_region_zone{$tablet->{tablet_uuid}}{$region}{$zone};
	$Tablet_bytes{ $tablet->{tablet_uuid} } = $tablet->{sst_size} + $tablet->{wal_size}; # Keep only ONE instance worth
	$tablet_bucket_max = $replicas if $replicas > $tablet_bucket_max;
}

sub Register_Region_Zone{ 
	my ($class, $region, $zone, $tserver_uuid) = @_;
	push @{ $region_zone{$region}{$zone}{TSERVER} }, $tserver_uuid; 
	$Tserver_to_region_zone{$tserver_uuid} = [$region, $zone];
}

sub Table_Report{ # CLass method
	print  "CREATE TABLE tableinfo(" 
	      , join(", ",  map {"$_ ". $field{$_}{TYPE} } sort {$field{$a}{SEQ} <=> $field{$b}{SEQ}} keys %field)
		  , ");\n";
	# Create temp table from a view, to make extraction faster ...
	print "CREATE TEMP TABLE temp_table_detail  AS SELECT * from table_detail;\n";
	
	for my $tkey (sort keys %collection){
	   my $t = $collection{$tkey};	
	   my $tablets_per_node = undef;
	   my $unbalanced="";
	   for (values %{$t->{NODE_TABLET_COUNT}}){
		      $tablets_per_node ||= $_;
			  if (abs($tablets_per_node - $_) > 1){
				  $unbalanced="(Unbalanced)";
			  }
	   }
       $t->{COMMENT}.=$unbalanced;
	   #UNIQ_tablet_count is not available YET. $t->{UNIQ_TABLET_COUNT} > $t->{UNIQ_TABLETS_ESTIMATE} and $t->{COMMENT}.="[Excess tablets]";
       
	   my $not_found_count = 0;
       for my $i (0..$t->{UNIQ_TABLETS_ESTIMATE} - 1){
		   #$i%10 == 0 and $t->{COMMENT} .= "[$i]";
		   $t->{KEYRANGELIST}[$i] and next;
		   $not_found_count++;
		   $not_found_count==1 and $t->{COMMENT}.="[".sprintf('0x%x',$i*$t->{KEYS_PER_TABLET})." \@$i not found]";
		   #$t->{COMMENT}.= ($t->{KEYRANGELIST}[$i]||0) .",";
	   }
	   $not_found_count>1 and $t->{COMMENT}.="[$not_found_count key ranges not found]";
	   
	   print "INSERT INTO tableinfo (",
	      , join(",",keys %field)
		  , ") values(\n   ",
		  , join (",", map({ my $x=$field{$_}{INSERT}; $x = $x ? $x->($t) : $t->{$_}; 
		                     $field{$_}{TYPE} eq "TEXT" ? "'" . $x . "'" : $x ||0
		                  } keys %field))
		  ,");\n";
	}
	
	print "DROP TABLE temp_table_detail;\n"; # No longer needed 
	print "UPDATE  tableinfo SET COMMENT=COMMENT || '[Excess tablets]'  WHERE UNIQ_TABLET_COUNT > UNIQ_TABLETS_ESTIMATE;\n";
	# Estimate the number of tablets per table that would result in <= 10GB tablets (for different n-node clusters)
	print << "__tablet_estimate__";
   CREATE VIEW large_tables AS 
   SELECT namespace,tablename,uniq_tablet_count as uniq_tablets,
      (sst_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as sst_RF1_mb,
	  (sst_tot_bytes /tot_tablet_count/1024/1024) as tablet_size_mb,
	  round((sst_tot_bytes*uniq_tablet_count/tot_tablet_count /1024.0/1024.0 + 5000) / 10000,1) as recommended_tablets,
	   tot_tablet_count / uniq_tablet_count as repl_factor,
      (wal_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as wal_RF1_mb
        FROM tableinfo
        WHERE sst_RF1_mb > 5000
        ORDER by sst_RF1_mb desc;

	CREATE VIEW table_sizes AS
	SELECT namespace, tablename, uniq_tablet_count as uniq_tablets,
		sst_tot_human as sst_bytes, 
		sst_rf1_human as sst_RF1_bytes,
		wal_tot_human as wal_bytes, 
		tot_human as total_bytes
	FROM tableinfo
	ORDER BY (sst_tot_bytes) DESC;

__tablet_estimate__

	# R e g i o n / Z o n e info

    print << "__region_zone_tablets__";

	CREATE TABLE region_zone_tablets(
		region		  TEXT,
		zone		  TEXT,
		tservers	  INTEGER,
		missing_replicas TEXT,
__region_zone_tablets__
    print  # Dynamically generate columns..
	       map ({"\[${_}_replicas\]   TEXT,\n"} 1..$tablet_bucket_max),
	      "balanced TEXT\n",
	      ");\n";
		

	for my $r (sort keys %region_zone){
		for my $z (sort keys %{ $region_zone{$r} }){
			#print "---REGION: $r  $z  ----- ", scalar(keys %{$region_zone{$r}{$z}{TABLET}}), " unique tablets ----\n";
			$Tablet_by_region_zone{$_}{$r}{$z} ||= 0 for keys %Tablet_by_region_zone;

			my ( $replica_count, $missing_bytes, @replica_bucket);
			for my $t (keys %Tablet_by_region_zone){
				my $replicas = $Tablet_by_region_zone{$t}{$r}{$z} ;
				if ($replicas == 0){
					$missing_bytes += $Tablet_bytes{$t};
				}
				$replica_bucket[ $replicas ]++;
				$replica_count++;
			}
			# Update missing bytes, and total tablets in the "1" bucket
			my @replica_bucket_text = map {$replica_bucket[$_] ||= 0} 0..$tablet_bucket_max ; # copy it , zeroing undefs
			$replica_bucket_text[0] = $replica_bucket[0] . sprintf(' (%.1f GB)',$missing_bytes/1024**3) if $missing_bytes;
			$replica_bucket_text[1] = $replica_bucket[1] . "/" . $replica_count;
			
			print "INSERT INTO region_zone_tablets VALUES('$r','$z',",
			      scalar(@{$region_zone{$r}{$z}{TSERVER}}), ",",    # Tserver count
                  map({"'" . $replica_bucket_text[$_] . "', "}0..$tablet_bucket_max),
                  ((grep {$replica_count == $replica_bucket[$_]} 1..$tablet_bucket_max) ? "'YES'": "'NO'"),
				  ");\n",
		} 
	}
};
1;	
} # End of TableInfo
####################################################################################
#--------------------------------------------------------------------------------
#NOTE:  The JSON output from the "-o json" option of yugatool is NOT WELL FORMED.
#       This package compensates for that - it parses pices collected by a higher level.
BEGIN{
package JSON_Analyzer;
# Package globals ---
my %dispatch_by_type = (
	cluster 		=> \&Parse_Cluster_line,
	Masters 		=> sub{Parse_Master_line($_[0],$_)  for @{$_[0]->{content}}},
	"Tablet Servers"=> sub{Parse_Tserver_line($_[0],$_) for @{$_[0]->{content}}},
	"Tablet Report" => sub{Parse_Tablet_line($_[0],$_)  for @{$_[0]->{content}}},
);

my $packed_zero = pack 'H4',0;      # Default value for start_key
my $packed_ffff = pack 'H4','ffff'; # ... and ed_key
my $json ; ## temp block = JSON->new;

sub Process_line{
   my ($j) = @_;

   #print "GOT ",length($j)," bytes at rec#$.\n";
   if ($json){
	   # All is well
   }else{
	  $json = entities_parser::->new()->{JSON};
   }

   	my $data = $json->decode($j);
    my $msg = $data->{msg} || "cluster"; # Initial JSON for cluster does not have "msg"
	$msg=~s/[^\w\s].*//; # Zap everything after text (Tablet report contains other stuff....
    ## print "-----", join(",",keys %$data),
    ##       " msg=",($msg),
    ##       " content=",scalar(@{$data->{content} || []}), "\n";
    ## 
	my $dispatch = $dispatch_by_type{$msg} or die "ERROR: No handler for $msg at $.";
	$dispatch->($data);
}

sub Parse_Cluster_line{
	my ($uuid,$zone) = @{$_[0]}{qw|clusterUuid zone|}; # No zone?
	$zone ||= "Version:" . $_[0]->{version};
	print "INSERT INTO cluster(type,uuid,zone) VALUES('CLUSTER',",
	       "'$uuid','$zone');\n";
	# "CLUSTER" is the default type of line, and only ONE such line should exist EARLY in the file.
	# Bail out If we find we are processing CLUSTER lines after 9 JSON records
    if ($. > 9){
	   die "ERROR: This does not appear to be a TABLET REPORT json (too many 'CLUSTER' lines)";	
	}
}

sub Parse_Master_line{
	my ($d,$m)=@_;
	my ($uuid, $host, $port, $region,$zone,$role) =
	    (MIME::Base64::decode_base64($m->{instanceId}{permanentUuid}),
		 $m->{registration}{privateRpcAddresses}[0]{host},
		 $m->{registration}{privateRpcAddresses}[0]{port},
		 $m->{registration}{cloudInfo}{placementRegion},
		 $m->{registration}{cloudInfo}{placementZone},
		 $m->{role}
		);
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,role)\n",
	      "  VALUES('MASTER','",
          join("','", $uuid,$host,$port,$region,$zone,$role),
		  "');\n";
}

sub Parse_Tserver_line{
	my ($d, $t) = @_;

	my ($uuid,$host,$port,$region,$zone,$alive,$reads,$writes,$heartbeat,$uptime,
        $sst_size,$sst_uncompressed,$sst_files,$memory) = (
		MIME::Base64::decode_base64($t->{instance_id}{permanentUuid}),
		$t->{registration}{common}{privateRpcAddresses}[0]{host},
		$t->{registration}{common}{privateRpcAddresses}[0]{port},
		$t->{registration}{common}{cloudInfo}{placementRegion},
		$t->{registration}{common}{cloudInfo}{placementZone},
        $t->{alive},
         undef, # reads
		 undef, # writes
		  undef, # heartbeat
		$t->{metrics}{uptimeSeconds},
		);
		;
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,uptime)\n",
	      "  VALUES('TSERVER','",
          join("','", $uuid,$host,$port,$region,$zone,$uptime),
		  "');\n";
	TableInfo::->Register_Region_Zone($region, $zone, $uuid);
}

sub Parse_Tablet_line{
	my ($d,$t) = @_;
	my ($host_name, $host_port,$host_uuid) 
	   = $d->{msg} =~m/\[host:"([^"]+)"\s+port:(\d+).*?\] \((\w+)/;

	my %values;
	@values{ qw|tablet_uuid tablename table_uuid namespace  state status  
	           start_key end_key         sst_size  wal_size 
				cterm cidx leader lease_status| }
			= (
			$t->{tablet}{tablet_status}{tabletId},
			$t->{tablet}{tablet_status}{tableName},
			$t->{tablet}{tablet_status}{tableId},
			$t->{tablet}{tablet_status}{namespaceName},
			$t->{tablet}{tablet_status}{state},
			$t->{tablet}{tablet_status}{tabletDataState},
			# start_key & end_key are base-64 encoded binary values 
			# They need to be decoded, then converted into ("0xABCD") "hex encoded binary"
			"0x". unpack('H4', MIME::Base64::decode_base64(
			   $t->{tablet}{tablet_status}{partition}{partitionKeyStart}) || $packed_zero),
			"0x". unpack('H4', MIME::Base64::decode_base64(
			   $t->{tablet}{tablet_status}{partition}{partitionKeyEnd})   || $packed_ffff),
	        $t->{tablet}{tablet_status}{sstFilesDiskSize},
			$t->{tablet}{tablet_status}{walFilesDiskSize},
			$t->{consensus_state}{cstate}{currentTerm},
			$t->{consensus_state}{cstate}{config}{opidIndex},
			$t->{consensus_state}{cstate}{leaderUuid},
			$t->{consensus_state}{leaderLeaseStatus},
	);

    print "INSERT INTO tablet (node_uuid,tablet_uuid , table_name,table_uuid,namespace,state,status,",
                  "start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status)\n  VALUES('",
				  $host_uuid,"'", 
	              map({ ",'" . ($values{$_}||'') . "'" } 
        		  qw|tablet_uuid tablename table_uuid namespace  state status  start_key end_key sst_size  wal_size 
             		  cterm cidx leader lease_status|  
		  ),");\n";
	TableInfo::find_or_new( \%values )
	        ->collect(\%values, $host_uuid);
}

} # ----- End of JSON_Analyzer ---------------------------------------

####################################################################################
BEGIN{
package entities_parser;

sub new{
	my ($class, %atts) = @_;
	my $self = bless {%atts}, $class;
	$self->{JSON_MODULE_EXISTS} =
		eval{
			require JSON;
			JSON->import();
			1;
		};

	if($self->{JSON_MODULE_EXISTS}){
		$self->{JSON} = JSON->new();
		# all well from this point on
	}else{
		die "ERROR: JSON (perl) module is not installed. Unable to process."
	}
	return $self;
}

sub Ingest_decode_and_Generate{
	my ($self) = @_;
 
	# Slurp file 
	my $contents = do {
		local $/;
		<>;
	};
    $self->{DECODED} = $self->{JSON}->decode($contents); 

    print << "__CREATE_TABLES__";
-- Generated by Entity parser 
CREATE TABLE ENT_KEYSPACE (id TEXT PRIMARY KEY,name,type);
CREATE TABLE ENT_TABLE (id TEXT PRIMARY KEY, keyspace_id,state, table_name);
CREATE TABLE ENT_TABLET (id TEXT ,table_id,state,is_leader,server_uuid,server_addr,type);

CREATE VIEW IF NOT EXISTS extra_Tablets_detail as 
    SELECT 'entities' as source, server_uuid as node, id as tablet from ENT_TABLET
    EXCEPT
    SELECT 'entities' as source,node_uuid,tablet_uuid FROM tablet
   UNION ALL
    SELECT 'tserv' as source, node_uuid,tablet_uuid FROM tablet
    EXCEPT
    SELECT 'tserv' as source, server_uuid as node,id as tablet from ENT_TABLET;

CREATE VIEW IF NOT EXISTS extra_Tablets_summary as 
   SELECT source,node,ip,region,count(*) FROM extra_Tablets_detail,cluster 
   WHERE node=uuid
   GROUP BY source,node,ip,region;
__CREATE_TABLES__

	main::Set_Transaction(1,"Entity Keyspaces");
	for my $ks (@{ $self->{DECODED}->{keyspaces} }){
		print "INSERT INTO ENT_KEYSPACE (id,name,type) VALUES('",
			join("','", map {$ks->{$_}} qw|keyspace_id keyspace_name keyspace_type|), "');\n";
	}
	main::Set_Transaction(0);

	main::Set_Transaction(1, "Entity TABLES");
	for my $tbl (@{ $self->{DECODED}->{tables} }){
		print "INSERT INTO ENT_TABLE (id, keyspace_id,state, table_name) VALUES('",
			join("','", map {$tbl->{$_}} qw|table_id keyspace_id state  table_name|), "');\n";
		}
	main::Set_Transaction(0);

	main::Set_Transaction(1,"Entity ENT_TABLETs");
	for my $t (@{ $self->{DECODED}->{tablets} }){
		my $leader = $t->{leader};
		for my $r (@{ $t->{replicas} }){
			print "INSERT INTO ENT_TABLET (id,table_id,state,is_leader,server_uuid,server_addr,type) VALUES('",
			
			join("','", $t->{tablet_id}, $t->{table_id}, $t->{state},
				($r->{server_uuid} eq $leader ? 1 : 0),
				map {$r->{$_}} qw|server_uuid addr type|), "');\n";
		}
	}
	main::Set_Transaction(0);
}
1;
} # End of entities_parser
####################################################################################
BEGIN{
package Tablet_Info;
# Parses output of yugatool "tablet_info"

{ #Local classes - pre-declaration
  package TSERVER;
  package TABLET;
}

sub new{

   print << "__CREATE_TABLES__";
CREATE TABLE IF NOT EXISTS tserver(id TEXT PRIMARY KEY,
            host TEXT ,placement_cloud TEXT ,placement_region TEXT ,
			placement_zone TEXT ,port TEXT);
CREATE TABLE IF NOT EXISTS TABLET_INFO ( UUID, server_uuid,last_status,
    leader_lease_status,namespace_name,op_id_index,op_id_term,sst_files_disk_size,
	state,table_id,table_name,tablet_data_state,leader);
CREATE VIEW IF NOT EXISTS good_tablets AS
    SELECT uuid,substr(server_uuid,1,5)||'..' as svr_uid,state,
	substr(tablet_data_state,-10) as data_state,substr(leader,1,5)||'..' as leader,
	substr(leader_lease_status,1,10) as ldr_lease,op_id_term, op_id_index,host as tserver
	FROM TABLET_INFO,tserver 
	WHERE  tablet_data_state !='TABLET_DATA_TOMBSTONED' and tserver.id=TABLET_INFO.server_uuid 
	ORDER BY UUID,op_id_term asc, op_id_index asc,server_uuid;

__CREATE_TABLES__

	my ($tserver,$tablet, $action, @stack);

	my %keyword_handler=(
		tablet_id		=> sub{$tablet = TABLET::->find_or_create($_[1] ."+". $tserver->{ID});
							$tablet->{UUID} = $_[1];
							$tablet->Dependent("TSERVER",$tserver->{ID});
							$tablet->{server_UUID} = $tserver->{ID};
							},
		map({$_=>sub{$tablet->{$_[0]}=$_[1]}} 
			qw| namespace_name table_name table_id state tablet_data_state last_status
					estimated_on_disk_size    consensus_metadata_disk_size 
				wal_files_disk_size sst_files_disk_size uncompressed_sst_files_disk_size 
				leader_lease_status
			|),
	);
	my %attrib_complete_action=(
		peers => sub {#print "Peer=",$_[0]->{permanent_uuid},"\n";
						my $ts = TSERVER::->find_or_create($_[0]->{permanent_uuid});
						$ts->populate_from_peer($_[0]);
						$_[0]=undef;
						$tablet->Dependent("TSERVER",$ts->{ID});
						},

		cstate => sub{$tablet->{LEADER} = $_[0]->{leader_uuid}},
		op_id  => sub{$tablet->{'op_id_' . $_} = $_[0]->{$_} for keys %{$_[0]}},
	);

	while(<>){
		next if m/^==/;
		next if m/^\s*$/; # empty
		next if m/^\d+:\s\d+$/; # 16:2
		
		if ( /\s*\[.+?\] \(UUID\s(\w+)\)/){  # [host:"240b:.." port:9100] (UUID 7a6..)"
		$tserver = TSERVER::->find_or_create($1);
		## $tablet and $tablet->Print();
		next;
		}
		if (m/^\s*}\s*$/){ # Close }
			my $completed = pop @stack;
			my $action = $attrib_complete_action{$completed};
			next unless $action;
			my $target = $tablet;
			for (@stack, $completed){ # include recently popped element 
					$target = $target->{$_}; 
			}
			$action->($target);
			next;
		}	
		if (my ($key,$val) = m/\s*(\w+):\s*(.+)/){
		$val =~tr/\"\r\n//d;
		if ($val eq "{"){
			push @stack, $key;
		}elsif (@stack){
			my $target = $tablet;
			for (@stack){
					$target = $target->{$_}||={}; 
			}
			$target->{$key} = $val;
			next;
		}
		my $action = $keyword_handler{$key} or next;

		$action->($key,$val,$_);
		next;	   
		}
	}

	## $tablet and $tablet->Print(); # Debug

	print "BEGIN TRANSACTION; -- $ARGV\n";

	TSERVER::->for_each(
		sub{
			my ($ts) = @_;
			$ts->Print_SQL();
		}
	);	

	TABLET::->for_each(
		sub{
			my ($t) = @_;
			$t->Print_SQL();
		}
	);

	print "COMMIT;\n";	
}


1;
} # End of Tablet_Info
#=============================================================================================
BEGIN{
package Generic::Class;
use Carp;

my %collection; # Index by {$class}{$id} get an instance

sub find_or_create{
  my ($class,$id,$att_ref) = @_;
  confess "ERROR: object ID required" unless $id;

  return $collection{$class}{$id}  if $collection{$class}{$id};
  #print "DEBUG:Creating object of class $class, id $id\n";
  return $collection{$class}{$id} = bless({ID=>$id, %{$att_ref||{}}}, $class);
}

sub Dependent{ # getter/setter
  my ($self,$dep_type,$dep_id,$dep_att) = @_;
  my $class = ref $self;
  my $dep = find_or_create($dep_type,$dep_id,$dep_att); 
  $self->{$dep_type}{$dep_id} and return $dep;
  $self->{DEP_COUNTS}{$dep_type}++;
  return $self->{$dep_type}{$dep_id} = $dep;
}

sub for_each_dependent{
  my ($self,$dep_type,$callback) = @_;
  confess  "ERROR: Callback method required" unless $callback and ref($callback) eq "CODE";
  $callback->(  $_ ) for keys %{$self->{$dep_type}};
}

sub for_each{
  my ($class,$callback) = @_;
  confess "ERROR: Callback method required" unless $callback and ref($callback) eq "CODE";
  $callback->(  $_ ) for values %{$collection{$class}};
}

sub Print{
	my ($self) = @_;
    print "OBJECT ",ref($self)," named $self->{ID} (";
	print join(",", map {"$_=$self->{$_}"} grep {!ref($self->{$_})}sort keys %$self),")\n";
    for my $dep_type (keys %{$self->{DEP_COUNTS}}){
	   print "\t$self->{DEP_COUNTS}{$dep_type} ${dep_type}'s :";
	   $self->for_each_dependent(
	     $dep_type,
	     sub{print $_[0]," "}
	   );
    }
    $self->{DEP_COUNTS} and print "\n";
}

sub Merge_Into{
   my ($self, $other) = @_;
   $other->{$_} ||= $self->{$_} for keys %$self; # Shallow copy
   $self->Delete();
   return $other;
}

sub Delete{
   my ($self) = @_;
   my $class = ref $self;
   delete $collection{$class}{$self->{ID}};
}	

} # End of generic class
#=============================================================================================
BEGIN{
package TSERVER;
use parent  -norequire, "Generic::Class";

my %collection; # Index by {$class}{$id} get an instance

sub populate_from_peer{
  my ($self,$peer) = @_;
  #print "Populating tserver\n";  
  $self->{$_} ||= $peer->{cloud_info}{$_} for keys %{ $peer->{cloud_info} };
  $self->{$_} ||= $peer->{last_known_private_addr}{$_} for keys %{ $peer->{last_known_private_addr} };
}

sub Print_SQL{
  my ($self) = @_;
  print "INSERT OR IGNORE INTO TSERVER (",
     "id, host  ,placement_cloud ,placement_region ,placement_zone ,port )",
        " VALUES (\n   '",
		join("','",map {$self->{$_}||''} qw |
		     ID host  placement_cloud placement_region placement_zone port
		   |),
        "');\n";		
}

1;
} # End of TSERVER
#======================================================================================

BEGIN{
package TABLET;

use parent  -norequire, "Generic::Class";

sub Print_SQL{
  my ($self) = @_;
  print "INSERT OR REPLACE  INTO TABLET_INFO (",
        " UUID, server_uuid,last_status,  leader_lease_status,namespace_name,op_id_index,op_id_term,",
		"sst_files_disk_size,state,table_id,table_name,tablet_data_state,leader)",
        " VALUES (\n   '",
		join("','",	map {$self->{$_}||''} qw |UUID server_UUID last_status 
    leader_lease_status namespace_name op_id_index op_id_term sst_files_disk_size 
	state table_id table_name tablet_data_state LEADER |),
        "');\n";		
}
1;
} # End of TABLET
#======================================================================================
INIT{
package MetricUnit;
	 # Use CLASS methods.  DO NOT INSTANTIATE.
use strict;
	# Local Class variables
	my $Kilo_Base = 1024; # Honest 2**10.
	my @ORDER= (" ",qw|K M G T P X Z Y| ); # Kilo, Meg, Gig, Tera,Peta
	my %MetricUnit;
	kilo_base($Kilo_Base); # Initialize %MetricUnit
	sub kilo_base{ #Get/Set Defaults to "binary (K=1024)"
	   my $new_base = shift;  # Could set to less honest K=1000.
	   return $Kilo_Base unless $new_base;
	   $Kilo_Base = $new_base;
	   # Initialize  Unit, Kilo, Mega Giga etc..(Y=2**80)
	   %MetricUnit = map {$ORDER[$_] => $Kilo_Base**$_ } 0..$#ORDER;
	   $MetricUnit{B} = 1; # Special case for BYTES
    }
	sub GetUnit{ # Convert to Human Readable (K,G etc)
		my ($number,$fixwidth) = @_;
		for my $power(reverse @ORDER){
			  next if $number < $MetricUnit{$power};
			  $number /= $MetricUnit{$power};
			  return ($number, $power eq ' '? '':$power); #Fix Empty Unit return
		 }
		 return ($number, $fixwidth < 0? "":" ");
    };
	sub format_kilo{  # Kilo, mega and gig
		my $number = shift || 0;
		my $fixwidth = shift;
		my $suffix ;
		($number,$suffix) = GetUnit($number, $fixwidth||0);
		# Split integer and decimal parts of the number
		my $integer = int($number);
		my $decimal = int(substr($number, length($integer)) * 10) # Max 1 decimal dig
				if (length($integer) < length($number));
		$decimal = '' unless defined $decimal ;#and $decimal > 0;
		# Combine integer and decimal parts and return the result.
		my $result = (length $decimal > 0 ?
					  join(".", $integer, $decimal) :
					  $integer);
	   		# Add Leading spaces if fixed width
		if ($fixwidth){
			if ($fixwidth > length($result)){
				$result =  ' ' x ($fixwidth - length($result) - length($suffix)) . $result;
			}else{ # need to truncate to integer part
				$result =  ' ' x ($fixwidth - length($integer) - length($suffix)) . $integer
		   }
		}
		# Combine it all back together and return it.
		return $result.$suffix;
	}
1;
} # End of Package MetricUnit
#=======================================================
#======================================================================================
