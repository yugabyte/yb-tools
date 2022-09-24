#!/usr/bin/perl
##########################################################################
## Tablet Report Parser

# Run instructions:

# 1: obtain a tablet report. To get that, run:

#		export MASTERS=10.183.11.69:7100,10.184.7.181:7100,10.185.8.17:7100
#		export TLS_CONFIG="--cacert /opt/yugabyte/certs/SchwabCA.crt"
#		export TLS_CONFIG="$TLS_CONFIG --skiphostverification"
#
#		./yugatool cluster_info \
#		  -m $MASTERS \
#		  $TLS_CONFIG \
#		  --tablet-report \
#		  > /tmp/tablet-report-$(hostname).out

# 2: Run THIS script  - which reads the tablet report and generates SQL.
#    Feed the generated SQL into sqlite3:

#        $ perl tablet_report_parser.pl < tablet-report-09-20T14.out | sqlite3 tablet-report-09-20T14.sqlite

# 3: Run Analysis using SQLITE ---

#         $ sqlite3 -header -column tablet-report-09-20T14.sqlite
#         SQLite version 3.31.1 2020-01-27 19:55:54
#         Enter ".help" for usage hints.
#         sqlite> select count(*) from leaderless;
#         count(*)
#         ----------
#         2321
#         sqlite> select *  from leaderless limit 3;
#         tablet_uuid                       table_name    node_uuid                         status                        ip
#         --------------------------------  ------------  --------------------------------  ----------------------------  -----------
#         67da88ffc8a54c63821fa85d82aaf463  custaccessid  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
#         31a7580dba224fcfb2cc57ec07aa056b  packetdocume  5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
#         9795475a798d411cb25c1627df13a122  packet        5a720d3bc58f409c9bd2a7e0317f2663  NO_MAJORITY_REPLICATED_LEASE  10.185.8.18
#         sqlite>


##########################################################################
our $VERSION = "0.06";
use strict;
use warnings;

my %opt=(
	STARTTIME	=>  scalar(localtime),
	DEBUG		=> 0,
	HOSTNAME    => $ENV{HOST} || $ENV{HOSTNAME} || $ENV{NAME} || qx|hostname|,
);
print << "__SQL__";
.print $0 Version $VERSION generating SQL on $opt{STARTTIME} 
CREATE TABLE cluster(type, uuid TEXT PRIMARY KEY, ip, port, region, zone ,role, uptime);
CREATE TABLE tablet (node_uuid,tablet_uuid TEXT , table_name,namespace,state,status,
                  start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status);
CREATE UNIQUE INDEX tablet_idx ON tablet (node_uuid,tablet_uuid);
CREATE VIEW tablets_per_table AS
     SELECT  table_name, count(*) as tablet_count, count(DISTINCT node_uuid) as nodes
	 FROM tablet GROUP BY table_name;
CREATE VIEW tablets_per_node AS
    SELECT node_uuid,min(ip) as node_ip,min(zone) as zone,  count(*) as tablet_count,
           count(DISTINCT table_name) as table_count	
	FROM tablet,cluster 
	WHERE cluster.type='TSERVER' and cluster.uuid=node_uuid 
	GROUP BY node_uuid
	ORDER BY tablet_count;
CREATE VIEW tablet_replica_detail AS
	SELECT tablet_uuid,count(*) as replicas  from tablet GROUP BY tablet_uuid;
CREATE VIEW tablet_replica_summary AS
	SELECT replicas,count(*) as tablet_count FROM  tablet_replica_detail GROUP BY replicas;
CREATE VIEW leaderless AS 
     SELECT t.tablet_uuid, replicas,table_name,node_uuid,status,ip 
	 from tablet t,cluster ,tablet_replica_detail trd
	 WHERE length(leader) < 3 AND cluster.type='TSERVER' AND cluster.uuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid;
-- table to handle hex values from 0x0000 to 0xffff
CREATE table hexval(h text primary key,i integer, covered integer);
WITH RECURSIVE
     cnt(x) AS (VALUES(0) UNION ALL SELECT x+1 FROM cnt WHERE x<0xffff)
    INSERT INTO hexval  SELECT printf('0x%0.4x',x) ,x, NULL  FROM cnt;
--- Summary report ----
CREATE VIEW summary_report AS
	SELECT (SELECT count(*) from cluster where type="TSERVER") || ' TSERVERs, '
			|| (SELECT count(DISTINCT table_name) FROM tablet) || " Tables, " 
			|| (SELECT count(*) from tablet) || ' Tablets loaded.' 
		AS Summary_Report
	UNION 
	 SELECT count(*) || ' leaderless tablets found.(See "leaderless")' from leaderless
	UNION
	 SELECT (SELECT sum(tablet_count) FROM tablet_replica_summary WHERE replicas < 3) 
			 || ' tablets have RF < 3. (See "tablet_replica_summary/detail")'
	;
CREATE VIEW version_info AS 
    SELECT '$0' as program, '$VERSION' as version, '$opt{STARTTIME}' AS run_on, '$opt{HOSTNAME}' as host;
__SQL__
#F[15] and next; m/\[ Tablet Report.*host:"([^"]+)".+\((\w+)\)/ and do{$node=$2;$host=$1}
#  ;$F[13] or next; print "$F[0],$node,$F[1],$F[13],\[$F[14]\],$host\n" '

my ( $line, $current_entity, $in_transaction);
my %entity = (
    CLUSTER => {REGEX=>'XXX-INVALID-NONEXISTANTXXX', HANDLER=>\&Parse_Cluster_line},
	MASTER  => {REGEX=>'^\[ Masters \]',       		 HANDLER=>\&Parse_Master_line},
	TSERVER => {REGEX=>'^\[ Tablet Servers \]',		 HANDLER=>\&Parse_Tserver_line },
	TABLET  => {REGEX=>'^\[ Tablet Report: ', 
	            HDR_EXTRACT => sub{$_[0] =~m/\[host:"([^"]+)" port:(\d+)\] \((\w+)/},
				HDR_KEYS    => [qw|HOST PORT NODE_UUID|],
				HANDLER=>\&Parse_Tablet_line,
				FIELD_LEN => [qw|35 63 13  10 20  12 10  11 11   8 11 35 55|], # unused
				LINE_REGEX =>
                 qr{\s(?<tablet_uuid>(\w{32}))\s{3}(?<tablename>(\w+))\s+(?<namespace>(\w+))\s+(?<state>(\w+))\s+(?<status>(\w+))\s+(?<start_key>(0x\w+)?)\s*(?<end_key>(0x\w+)?)\s*(?<sst_size>(\d+\s\w+))\s+(?<wal_size>(\d+\s\w+))\s+\s*(?<cterm>(\d+))\s*(?<cidx>(\d+))\s+(?<leader>([\[\]\w]+))\s+(?<lease_status>(\w+)?)},
				 
	},
);
my $entity_regex = join "|", map {$entity{$_}{REGEX}} keys %entity;
$current_entity = "CLUSTER"; # This line should have been in the report, but first item is the cluster 
while ($line=<>){
	if (length($line) < 5){ # Blank line 
	   #causes problems# $current_entity = undef;	
	   next;
	}
	if ($line =~m/$entity_regex/){
		print ".print  ... $entity{$current_entity}{COUNT} $current_entity items processed.\n"; # For previous enity 
	    ($current_entity) = grep {$line =~m/$entity{$_}{REGEX}/ } keys %entity;
		print ".print Processing line# $. :$current_entity from $line";
		$entity{$current_entity}{COUNT} = 0;
		Set_Transaction(1,"Starting $current_entity");
		next unless my $extract_sub = $entity{$current_entity}{HDR_EXTRACT};
		my @extracted_values = $extract_sub->($line);
		$entity{$current_entity}{$_} = shift @extracted_values for @{$entity{$current_entity}{HDR_KEYS}};
		print "--     for ", map({"$_=$entity{$current_entity}{$_} "} @{$entity{$current_entity}{HDR_KEYS}} ),"\n";
		next;
	}
    if (substr($line,0,3) eq "   " ){
		Process_Headers();
		print "--Headers:", map({$_->{NAME} . "(",$_->{START},"), "} @{$entity{$current_entity}{HEADERS}}),"\n";
		next;
	}
    die "ERROR:Line $.: No context for $line" unless $current_entity;
	$entity{$current_entity}{HANDLER}->(); # $line is global
	$entity{$current_entity}{COUNT}++;
}
Set_Transaction(0);
my $tmpfile = "/tmp/tablet-report-analysis-settings$$";

print << "__ENDING_STUFF__";
.print  ... $entity{$current_entity}{COUNT} $current_entity items processed.
.print SQL loading Completed.
.print --- Available REPORT-NAMEs ---
.tables
.print --- Summary Report ---
SELECT '     ',* FROM summary_report;

-- .print DEBUG output file is $tmpfile 
.output $tmpfile
.show
.output
.print --- To get a report, run: ---
--  Note: strange escaping below for the benefit of perl, sqlite, and the shell 
.shell perl -nE 'm/filename: (\\S+)/ and say qq^\\\tsqlite3 -header -column \\\$1 \\"SELECT \\* from REPORT-NAME\\"^'  $tmpfile
.shell rm $tmpfile
__ENDING_STUFF__

exit 0;

#-------------------------------------------------------------------------------------
sub Parse_Cluster_line{
	my ($uuid,$zone) = $line=~m/^\s*([\w\-]+).+(\[.*\])/;
	print "INSERT INTO cluster(type,uuid,zone) VALUES('CLUSTER',",
	       "'$uuid','$zone');\n";
    if ($. > 3){
	   die "ERROR: This does not appear to be a TABLET REPORT (too many 'CLUSTER' lines)";	
	}
}
sub Parse_Master_line{
	my ($uuid, $host, $port, $region,$zone,$role) = $line=~m/(\S+)/g;
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,role)\n",
	      "  VALUES('MASTER','",
          join("','", $uuid,$host,$port,$region,$zone,$role),
		  "');\n";
}

sub Parse_Tserver_line{
	my ($uuid,$host,$port,$region,$zone,$alive,$reads,$writes,$heartbeat,$uptime,
        $sst_size,$sst_uncompressed,$sst_files,$memory)
		= split /\s\s+/,$line ;
	$uuid =~s/^\s+//g; #Zap leading space
	print "INSERT INTO cluster(type, uuid , ip, port, region, zone ,uptime)\n",
	      "  VALUES('TSERVER','",
          join("','", $uuid,$host,$port,$region,$zone,$uptime),
		  "');\n";
    $entity{TSERVER}{BY_UUID}{$uuid}={
		HOST=>$host,PORT=>$port,REGION=>$region,ZONE=>$zone,UPTIME=>$uptime
	        };
}

sub Parse_Tablet_line{

# 0a2aa531ce7541f4bfffc634200d16c5   brokerageaccountphone                                          titan_prod   RUNNING   TABLET_DATA_READY   0x728e      0x7538    21 MB      2048 kB    27      288541     b686d09824b4455997873522dedcd3a9   HAS_LEASE
	##my ($tablet,$table,$namespace,$state,$status,$start_key,$end_key,
	##    $sst_size,$wal_size,$cterm,$cidx,$leader,$lease_status)
	##	= unpack("x1 A32 x3 A63 A13  A10 A20  A12 A10  A11 A11   A8 A11 A35 A55",$line);
	##if ($table =~/\s/){
	##   # We have mis-parsed this line (Offsets are not what we expected.
	##   print "--Line $. unpack error. table=$table\n";
	##}		
    if ($line =~ $entity{$current_entity}{LINE_REGEX}){
		# Fall through and process it
	}else{
	    #Regex failed to match 
         die "ERROR: Line $. failed to match tablet regex";		
	}
    ##print "INSERT INTO tablet (tablet_uuid,table_name,node_uuid, leader,status) VALUES('",
	##     join("','",$tablet, $table, $entity{TABLET}{NODE_UUID}, $leader, $lease_status),
	##	 "');\n";

	print "INSERT INTO tablet (node_uuid,tablet_uuid , table_name,namespace,state,status,",
                  "start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status) VALUES('",
				  $entity{TABLET}{NODE_UUID},"'", 
	              map({ ",'" . $+{$_} . "'" } 
        		  qw|tablet_uuid tablename namespace  state status  start_key end_key sst_size  wal_size 
             		  cterm cidx leader lease_status|  
		  ),");\n";
	my %save_val=%+; # Save collected regex named capture hash (before it gets clobbered by next regex)
	if ( $save_val{start_key} and !$save_val{end_key}
	    and $line =~/\s{12}$save_val{start_key}   /
	){
	   # Special case - in $line, $start key is missing, but regex mistakenly places end at start.
	   print "UPDATE tablet SET start_key='', end_key='$save_val{start_key}' ",
	         " WHERE tablet_uuid='$save_val{tablet_uuid}' AND node_uuid='",
			 $entity{TABLET}{NODE_UUID}, "'; -- correction for line $.\n";
       	   
	}
	#print "INSERT INTO tablet (node_uuid,tablet_uuid , table_name,namespace,state,status,",
    #              "start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status) VALUES('",
	#     join("','",$entity{TABLET}{NODE_UUID},$tablet, $table, $namespace,$state,$status,
    #                $start_key, $end_key, $sst_size, $wal_size, $cterm, $cidx,$leader, $lease_status),
	#	 "');\n";	 
}

sub Process_Headers{
		$entity{$current_entity}{PREVIOUS_HEADERS} = $entity{$current_entity}{HEADERS};
		$entity{$current_entity}{HEADERS}=[]; # Zap it 
		my $hdr_idx = 0;
		while ( $line =~/([A-Z_]+)/g ){
			my $hdr_item = $1;
			
		   	$entity{$current_entity}{HEADERS}[$hdr_idx] =  {NAME=>$hdr_item, START=>$-[0], END=> $+[0], LEN=> $+[0] - $-[0]};
			#if ($entity{$current_entity}{HEADERS}[$hdr_idx]{START} != ($entity{$current_entity}{PREVIOUS_HEADERS}[$hdr_idx]{START}||=0)){
			#	print "-- Hdr: ",$entity{$current_entity}{HEADERS}[$hdr_idx]{NAME} , " changed from ",
			#	      $entity{$current_entity}{PREVIOUS_HEADERS}[$hdr_idx]{START}," to ",$entity{$current_entity}{HEADERS}[$hdr_idx]{START},"\n";
		    #}
			$hdr_idx++;
		}
}

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
