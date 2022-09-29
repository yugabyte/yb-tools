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
our $VERSION = "0.08";
use strict;
use warnings;

BEGIN{ # Namespace declaration
  package TableInfo;
} # End of namespace declaration 
my %opt=(
	STARTTIME	=>  scalar(localtime),
	DEBUG		=> 0,
	HOSTNAME    => $ENV{HOST} || $ENV{HOSTNAME} || $ENV{NAME} || qx|hostname|,
);
print << "__SQL__";
.print $0 Version $VERSION generating SQL on $opt{STARTTIME} 
CREATE TABLE cluster(type, uuid TEXT PRIMARY KEY, ip, port, region, zone ,role, uptime);
CREATE TABLE tablet (node_uuid,tablet_uuid TEXT , table_name,table_uuid, namespace,state,status,
                  start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status);
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
	SELECT namespace,table_name,table_uuid,tablet_uuid,count(*) as replicas  
	from tablet 
	GROUP BY namespace,table_name,table_uuid,tablet_uuid;
CREATE VIEW tablet_replica_summary AS
	SELECT replicas,count(*) as tablet_count FROM  tablet_replica_detail GROUP BY replicas;
CREATE VIEW leaderless AS 
     SELECT t.tablet_uuid, replicas,t.table_name,node_uuid,status,ip 
	 from tablet t,cluster ,tablet_replica_detail trd
	 WHERE length(leader) < 3 AND cluster.type='TSERVER' AND cluster.uuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid;
CREATE VIEW delete_laderless_be_careful AS 
     SELECT '\$HOME/tserver/bin/yb-ts-cli delete_tablet '|| tablet_uuid ||' -certs_dir_name \$TLSDIR -server_address '||ip ||':9100  Your_REASON_tktnbr'
	   AS generated_delete_command
     FROM leaderless;
-- table to handle hex values from 0x0000 to 0xffff (Not requird) 
--CREATE table hexval(h text primary key,i integer, covered integer);
--WITH RECURSIVE
--     cnt(x) AS (VALUES(0) UNION ALL SELECT x+1 FROM cnt WHERE x<0xffff)
--    INSERT INTO hexval  SELECT printf('0x%0.4x',x) ,x, NULL  FROM cnt;
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
	            HDR_EXTRACT => sub{$_[0] =~m/\[host:"([^"]+)"\s+port:(\d+)\] \((\w+)/},
				HDR_KEYS    => [qw|HOST PORT NODE_UUID|],
				HANDLER=>\&Parse_Tablet_line,
				LINE_REGEX =>
                 	qr| ^\s(?<tablet_uuid>(\w{32}))\s{3}
					(?<tablename>(\w+))\s+
					(?<table_uuid>(\w{32})?)\s* # This exists only if --show_table_uuid is set
					(?<namespace>(\w+))\s+
					(?<state>(\w+))\s+
					(?<status>(\w+))\s+
					(?<start_key>(0x\w+)?)\s* # This could be EMPTY. If so, start_key value will contain END!
					(?<end_key>(0x\w+)?)\s*   # This could also be EMPTY, but start exists in that case
					(?<sst_size>(\d+\s\w+))\s+
					(?<wal_size>(\d+\s\w+))\s+
					(?<cterm>(\d+))\s*
					(?<cidx>(\d+))\s+
					(?<leader>([\[\]\w]+))\s+
					(?<lease_status>(\w+)?)|x,
									 
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
		chomp $line;
		print ".print Processing $current_entity from $line (Line#$.)\n";
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
print << "__MAIN_COMPLETE__";
.print  ... $entity{$current_entity}{COUNT} $current_entity items processed.
.print Main SQL loading Completed. Generating table stats...
__MAIN_COMPLETE__

Set_Transaction(1);
TableInfo::Table_Report();
Set_Transaction(0);

my $tmpfile = "/tmp/tablet-report-analysis-settings$$";

print << "__ENDING_STUFF__";
.print --- Completed. Available REPORT-NAMEs ---
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
	if (substr($line,0,1) eq "{"){ # Some sort of error message - ignore
	    chomp $line;
		print ".print ERROR in input line# $. : ",substr($line,0,40)," ... ignored.\n";
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
         die "ERROR: Line $. failed to match tablet regex";		
	}

	print "INSERT INTO tablet (node_uuid,tablet_uuid , table_name,table_uuid,namespace,state,status,",
                  "start_key, end_key, sst_size, wal_size, cterm, cidx, leader, lease_status) VALUES('",
				  $entity{TABLET}{NODE_UUID},"'", 
	              map({ ",'" . ($+{$_}||'') . "'" } 
        		  qw|tablet_uuid tablename table_uuid namespace  state status  start_key end_key sst_size  wal_size 
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
	UNIQ_TABLET_COUNT=>  {TYPE=>'INTEGER',VALUE=>0,  INSERT=>sub{return "(SELECT unique_tablet_count from table_detail WHERE table_name='" 
	                                                             . $_[0]->{TABLENAME} . "' and namespace='"
																 . $_[0]->{NAMESPACE} . "')"}
						, SEQ=>5} ,
	KEYRANGELIST	=>  {TYPE=>'INTEGER',VALUE=>[], INSERT=>sub{return scalar(@{ $_[0]->{KEYRANGELIST} })}, SEQ=>6} ,
	UNIQ_TABLETS_ESTIMATE =>  {TYPE=>'INTEGER',VALUE=>0,  INSERT=>sub{sprintf '%.2f',$_[0]->{UNIQ_TABLETS_ESTIMATE} }, SEQ=>7},
	NODE_TABLET_MIN  => {TYPE=>'INTEGER',VALUE=>{}, INSERT=>sub{my $n=999999999; $_ < $n? $n=$_:0 for values %{$_[0]->{NODE_TABLET_COUNT}}; $n}, SEQ=>8} ,
	NODE_TABLET_MAX  => {TYPE=>'INTEGER',VALUE=>{}, INSERT=>sub{my $n=0; $_ > $n? $n=$_:0 for values %{$_[0]->{NODE_TABLET_COUNT}}; $n}, SEQ=>9} ,
    KEYS_PER_TABLET	 => {TYPE=>'INTEGER',VALUE=>0, SEQ=>10 },
    UNMATCHED_KEY_SIZE=>{TYPE=>'INTEGER',VALUE=>0, SEQ=>11 },	

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
	my $start_key = hex($tablet->{start_key} || '0x0000'); # Convert to a binary number 
	my $end_key   = hex($tablet->{end_key}   || '0xffff');
	
	if (0 == ($self->{UNIQ_TABLETS}||=0)){
	   # Need to calcuate this 	
	   $self->{KEYS_PER_TABLET}        = $end_key - $start_key + 1; 
	   $self->{UNIQ_TABLETS_ESTIMATE} = (0xffff + 1) / $self->{KEYS_PER_TABLET} ; 
	}
	if (($end_key - $start_key) ==  $self->{KEYS_PER_TABLET}){
		# Matches previous tablets .. all is well
	}else{
		#print ".print ERROR:Line $.: Tablet $tablet->{tablet_uuid} offsets $end_key - $start_key dont match diff=$self->{KEYS_PER_TABLET}\n";
		$self->{UNMATCHED_KEY_SIZE}++;
	}
	$self->{KEYRANGELIST}[$start_key / $self->{KEYS_PER_TABLET} ] ++;
}

sub Table_Report{ # CLass method
    #CREATE TABLE tableinfo(table_name,table_uuid, namespace, tablets_unique INTEGER, tablets_replicas INTEGER, 
    #               tablet_rf_min INTEGER, tablet_rf_max INTEGER,all_keys_prsent INTEGER, comment TEXT);
    #CREATE UNIQUE INDEX tableinfo_key ON tableinfo (namespace,table_name,table_uuid);
	print  "CREATE TABLE tableinfo(" 
	      , join(", ",  map {"$_ ". $field{$_}{TYPE} } sort {$field{$a}{SEQ} <=> $field{$b}{SEQ}} keys %field)
		  , ");\n";
	
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
	   #print STDERR  "-- Table $tkey \t$t->{TABLETCOUNT} tablets+replicas $unbalanced, "
	   #    ,sprintf('%d',$t->{UNIQ_TABLETS})," Uniq tablets per table, "
	   #    ,scalar(@{$t->{KEYRANGELIST}})," slots, "
		#	,$t->{KEYS_PER_TABLET}  ," Keys per tablet:\n";
	  ## print "INSERT INTO tableinfo(table_name,table_uuid, namespace, tablets_unique , tablets_replicas , \n",
      ##       "        tablet_rf_min , tablet_rf_max ,all_keys_prsent ,comment) values(\n",
	  ##       map({ "'" . ($t->{$_} || ""). "',"} qw|TABLENAME TABLE_UUID  NAMESPACE| ), "\n     ",
		##	 map({ ($t->{$_} || 0). ","} qw|UNIQ_TABLETS TABLETCOUNT | ), "\n     ",
	  ##       ",0,0,0,'$unbalanced test data');"
	  ## ;
       my $out="\t";
       for my $i (0..$t->{UNIQ_TABLETS_ESTIMATE} - 1){
		   $i%10 == 0 and $out .= "[$i]";
		  $out.= ($t->{KEYRANGELIST}[$i]||0) .",";
	   }
	   #print STDERR "$out\n";
	   print "INSERT INTO tableinfo (",
	      , join(",",keys %field)
		  , ") values(\n   ",
		  , join (",", map({ my $x=$field{$_}{INSERT}; $x ? $x->($t) :  
		                     $field{$_}{TYPE} eq "TEXT" ? "'" . $t->{$_} . "'" : $t->{$_}||0       
		                  } keys %field))
		  ,");\n";
	}
	
};
1;	
} # End of TableInfo
####################################################################################
