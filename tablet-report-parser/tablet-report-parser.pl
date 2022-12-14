#!/usr/bin/perl
##########################################################################
## Tablet Report Parser

# Run instructions:

# 1: obtain a tablet report. To get that, run Yugatool;:
# (See https://support.yugabyte.com/hc/en-us/articles/6111093115405-How-to-use-the-Yugatool-utility)
#
#		export MASTERS=10.183.11.69:7100,10.184.7.181:7100,10.185.8.17:7100
#		export TLS_CONFIG="--cacert /opt/yugabyte/certs/SchwabCA.crt"
#		export TLS_CONFIG="$TLS_CONFIG --skiphostverification"
#
#		./yugatool cluster_info \
#		  -m $MASTERS \
#		  $TLS_CONFIG \
#		  --tablet-report \
#		  > /tmp/tablet-report-$(hostname).out
#
#       You can also use the "-o json" option. This code accepts that output also.

# 2: Run THIS script  - which reads the tablet report and generates SQL.
#    Feed the generated SQL into sqlite3:

#        $ perl tablet_report_parser.pl tablet-report-09-20T14.out | sqlite3 tablet-report-09-20T14.sqlite

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
our $VERSION = "0.21";
use strict;
use warnings;
#use JSON qw( ); # Older systems may not have JSON, invoke later, if required.
use MIME::Base64;

BEGIN{ # Namespace forward declarations
  package TableInfo;
  package JSON_Analyzer;
} # End of namespace declaration 
my %opt=(
	STARTTIME	=>  scalar(localtime),
	DEBUG		=> 0,
	HOSTNAME    => $ENV{HOST} || $ENV{HOSTNAME} || $ENV{NAME} || qx|hostname|,
	JSON        => 0, # Auto set to 1 when  "JSON" discovered. default is Reading "table" style.
);
# Sqlite 3.7 does not support ".print", so we use wierd SELECT statements to print messages.
print << "__SQL__";
SELECT '$0 Version $VERSION generating SQL on $opt{STARTTIME}';
CREATE TABLE cluster(type, uuid TEXT PRIMARY KEY, ip, port, region, zone ,role, uptime);
CREATE TABLE tablet (node_uuid,tablet_uuid TEXT , table_name,table_uuid, namespace,state,status,
                  start_key, end_key, sst_size INTEGER, wal_size INTEGER, cterm, cidx, leader, lease_status);
CREATE UNIQUE INDEX tablet_idx ON tablet (node_uuid,tablet_uuid);

CREATE VIEW table_detail AS
     SELECT  namespace,table_name, count(*) as total_tablet_count,count(DISTINCT tablet_uuid) as unique_tablet_count, count(DISTINCT node_uuid) as nodes
	 FROM tablet GROUP BY namespace,table_name;
CREATE VIEW tablets_per_node AS
    SELECT node_uuid,min(ip) as node_ip,min(zone) as zone,  count(*) as tablet_count,
           count(DISTINCT table_name) as table_count	
	FROM tablet,cluster 
	WHERE cluster.type='TSERVER' and cluster.uuid=node_uuid 
	GROUP BY node_uuid
	ORDER BY tablet_count;
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

-- table to handle hex values from 0x0000 to 0xffff (Not requird) 
--CREATE table hexval(h text primary key,i integer, covered integer);
--WITH RECURSIVE
--     cnt(x) AS (VALUES(0) UNION ALL SELECT x+1 FROM cnt WHERE x<0xffff)
--    INSERT INTO hexval  SELECT printf('0x%0.4x',x) ,x, NULL  FROM cnt;
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
	;
CREATE VIEW version_info AS 
    SELECT '$0' as program, '$VERSION' as version, '$opt{STARTTIME}' AS run_on, '$opt{HOSTNAME}' as host;
__SQL__

my ( $line, $json_line, $current_entity, $in_transaction);
my %entity = (
    REPORTINFO => {REGEX => '^\[ ReportInfo \]', HANDLER=>sub{print "--$.:$line\n"}, COUNT=>0},
    CLUSTER => {REGEX=>'^\[ Cluster \]', HANDLER=>\&Parse_Cluster_line,              COUNT=>0},
	MASTER  => {REGEX=>'^\[ Masters \]',       		 HANDLER=>\&Parse_Master_line},
	TSERVER => {REGEX=>'^\[ Tablet Servers \]',		 HANDLER=>\&Parse_Tserver_line },
	TABLET  => {REGEX=>'^\[ Tablet Report: ', 
	            HDR_EXTRACT => sub{$_[0] =~m/\[host:"([^"]+)"\s+port:(\d+)\] \((\w+)/},
				HDR_KEYS    => [qw|HOST PORT NODE_UUID|],
				HANDLER=>\&Parse_Tablet_line,
				LINE_REGEX =>
                 	qr| ^\s(?<tablet_uuid>(\w{32}))\s{3}
					(?<tablename>([\w\-]+))\s+
					(?<table_uuid>(\w{32})?)\s* # This exists only if --show_table_uuid is set
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
my %kilo_multiplier=(
    BYTES	=> 1,
	KB		=> 1024,
	MB		=> 1024*1024,
	GB		=> 1024*1024*1024,
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
		   print "SELECT '.print JSON input detected.';\n";
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
# --- End of Main loop ----
Set_Transaction(0);

if ($opt{JSON}){
	# no table report
}else{
	print << "__MAIN_COMPLETE__";
SELECT '  ... $entity{$current_entity}{COUNT} $current_entity items processed.';
SELECT 'Main SQL loading Completed. Generating table stats...';
__MAIN_COMPLETE__
	Set_Transaction(1);
	TableInfo::Table_Report();
	Set_Transaction(0);
}
my $tmpfile = "/tmp/tablet-report-analysis-settings$$";

print << "__ENDING_STUFF__";
SELECT '--- Completed. Available REPORT-NAMEs ---';
.tables
.output $tmpfile
.databases
.output stdout
SELECT '','--- Summary Report ---'
UNION 
SELECT '     ',* FROM summary_report;
.quit
__ENDING_STUFF__
close STDOUT; # Done talking to sqlite3
open  STDOUT, ">", "/dev/null"; # To avoid Warning about "Filehandle STDOUT reopened as xxx"
my $retry = 30; # Could take 30 sec to generate summary report...
while ($retry-- > 0 and ! -e $tmpfile){
  sleep 1; # Wait for sqlite to close up 
}
#-- OLD CODE SQL commented below
#-- .show
#-- .output stdout
#-- .print --- To get a report, run: ---
#-- -- Note: strange escaping below for the benefit of perl, sqlite, and the shell. Wierdness just to get sqlite file name.
#-- .shell perl -nE 'm/filename: (\\S+)/ and say qq^\\\tsqlite3 -header -column \\\$1 \\"SELECT \\* from REPORT-NAME\\"^'  $tmpfile
#-- .shell rm $tmpfile

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
sub Parse_Cluster_line{
	my ($uuid,$zone) = $line=~m/^\s*([\w\-]+).+(\[.*\])/; 
	if (! $zone){ # Zone may not have been enclosed in []
	   my @piece = split /\s+/,	$line;
	   $piece[0] eq '' and shift @piece; # First piece is empty because of leading blanks in $line 
	   $uuid = $piece[0];
	   my ($zone_idx) = grep { $entity{$current_entity}{HEADERS}[$_]->{NAME} eq "ZONES" } 0..$#{ $entity{$current_entity}{HEADERS} };
	   $zone = $piece[$zone_idx];
	}
	print "INSERT INTO cluster(type,uuid,zone) VALUES('CLUSTER',",
	       "'$uuid','$zone');\n";
    if ($. > 9){
	   print "SELECT 'ERROR: This does not appear to be a TABLET REPORT (too many CLUSTER lines)';\n";	
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
    $entity{TSERVER}{BY_UUID}{$uuid}={
		HOST=>$host,PORT=>$port,REGION=>$region,ZONE=>$zone,UPTIME=>$uptime
	        };
}

sub Parse_Tablet_line{

# 0a2aa531ce7541f4bfffc634200d16c5   brokerageaccountphone                                          titan_prod   RUNNING   TABLET_DATA_READY   0x728e      0x7538    21 MB      2048 kB    27      288541     b686d09824b4455997873522dedcd3a9   HAS_LEASE

    if ($line =~ $entity{$current_entity}{LINE_REGEX}){
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
					(?<tablename>(\w+))\s+
					(?<table_uuid>(\w{32})?)\s* # This exists only if --show_table_uuid is set
					# NOTE: <namespace> has been REMOVED from this regex
					(?<state>(\w+))\s+
					(?<status>(\w+))\s+
					(?<start_key>(0x\w+)?)\s* 
					(?<end_key>(0x\w+)?)\s*  
					(?<sst_size>(\-?\d+))  \s  (?<sst_unit>(\w+)) \s+  # "0 bytes"|"485 kB"|"12 MB"
					(?<wal_size>(\-?\d+))  \s  (?<wal_unit>(\w+)) \s+
					(?<cterm>([\[\]\d]+))\s*    # This could be "[]" or a number.. 
					(?<cidx>([\[\]\d]+))\s+
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
				  $entity{TABLET}{NODE_UUID},"'", 
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
			 $entity{TABLET}{NODE_UUID}, "'; -- correction for line $.\n";
       $save_val{end_key}   = $save_val{start_key};
       $save_val{start_key} = undef;	   
	}
	
    TableInfo::find_or_new( \%save_val )
	        ->collect(\%save_val, $entity{TABLET}{NODE_UUID});
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
####################################################################################
BEGIN{
package TableInfo;

my %collection = (); # Collection of Tableinfo objects, key is namespace:table_name:uuid  
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
	
	if (0 == ($self->{UNIQ_TABLETS_ESTIMATE}||=0)){
	   # Need to calcuate this 	
	   $self->{KEYS_PER_TABLET}        = $end_key - $start_key; # keys < $end key, so don't add 1. 
	   $self->{UNIQ_TABLETS_ESTIMATE} = int( (0xffff) / $self->{KEYS_PER_TABLET} ); # Truncate decimals 
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
	$start_key % $self->{KEYS_PER_TABLET} != 0 and $self->{KEY_RANGE_OVERLAP}++; 
	$self->{KEYRANGELIST}[int($start_key / $self->{KEYS_PER_TABLET}) ] ++;
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
		  , join (",", map({ my $x=$field{$_}{INSERT}; $x ? $x->($t) :  
		                     $field{$_}{TYPE} eq "TEXT" ? "'" . $t->{$_} . "'" : $t->{$_}||0       
		                  } keys %field))
		  ,");\n";
	}
	
	print "DROP TABLE temp_table_detail;\n"; # No longer needed 
	print "UPDATE  tableinfo SET COMMENT=COMMENT || '[Excess tablets]'  WHERE UNIQ_TABLET_COUNT > UNIQ_TABLETS_ESTIMATE;\n";
	# Estimate the number of tablets per table that would result in <= 10GB tablets (for different n-node clusters)
	print << "__tablet_estimate__";
   CREATE VIEW large_tables AS 
   SELECT namespace,tablename,uniq_tablet_count as uniq_tablets,
      (sst_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as sst_table_mb,
	  (sst_tot_bytes /tot_tablet_count/1024/1024) as tablet_size_mb,
	  round((sst_tot_bytes /1024.0/1024.0/8.0  + 5000) / 10000,1) as rec_8node_tablets,
	  round((sst_tot_bytes /1024.0/1024.0/12.0 + 5000) / 10000,1) as rec_12node_tablets,
	  round((sst_tot_bytes /1024.0/1024.0/24.0 + 5000) / 10000,1) as rec_24node_tablets,
	   tot_tablet_count / uniq_tablet_count as repl_factor,
      (wal_tot_bytes)*uniq_tablet_count/tot_tablet_count/1024/1024 as wal_table_mb
        FROM tableinfo
        WHERE sst_table_mb > 5000
        ORDER by sst_table_mb desc;

__tablet_estimate__
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
	    my $json_module_exists =
     	  eval{
		    require JSON;
		    JSON->import();
		    1;
		  };

	    if($json_module_exists){
	        $json = JSON->new();
			# all well from this point on
	    }else{
	        die "ERROR: JSON (perl) module is not installed. Unable to process."
	    }
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
    if ($. > 5){
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
}

sub Parse_Tablet_line{
	my ($d,$t) = @_;
	my ($host_name, $host_port,$host_uuid) 
	   = $d->{msg} =~m/\[host:"([^"]+)"\s+port:(\d+)\] \((\w+)/;

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
}

} # ----- ENd of JSON_Analyzer ---------------------------------------

