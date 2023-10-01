#!/usr/bin/perl

our $VERSION = "0.09";
my $HELP_TEXT = << "__HELPTEXT__";
    It's a me, moses.pl  Version $VERSION
               ========
 Get and analyze info on all tablets in the system.
 By default, output will be piped to sqlite3 to create a sqlite3 database,
 and run default reports.
 
 Run options:
   --YBA_HOST         [=] <YBA hostname or IP> (Required)
   --API_TOKEN        [=] <API access token>   (Required)
   --UNIVERSE         [=] <Universe Name>  (Required. Can be partial, sufficient to be Unique)
   --GZIP             (Use this if you want to create a sql.gz for export, instead of a sqlite DB)
   **ADVANCED options**
   --HTTPCONNECT      [=] [curl | tiny]    (Optional. Whether to use 'curl' or HTTP::Tiny(Default))
   --FOLLOWER_LAG_MINIMUM [=] <value> (ms)(collect tablet follower lag for values >= this value(default 1000))
   
    If STDOUT is redirected, it can be sent to  a SQL file, or gzipped, and collected for offline analysis.
__HELPTEXT__
use strict;
use warnings;
use Getopt::Long;
use HTTP::Tiny;
use POSIX;

{
 package Tablet;
 package Web::Interface;
 package JSON::Tiny;
 package DatabaseClass;
}; # Pre-declare local modules 

my %opt = (
   STARTTIME_PRINTABLE  => unixtime_to_printable(time(),"YYYY-MM-DD HH:MM","Include tz offset"),
   STARTTIME            => time(),
   CURRENT_TIME         => time(),
   DEBUG                => 0,
   HELP                 => 0,
   LOCALHOST            => ($ENV{HOSTNAME} || do{chomp(local $_=qx|hostname|); $_}),
   API_TOKEN            => undef,
   YBA_HOST             => undef,
   #CUST_UUID            => undef,
   #UNIV_UUID            => undef,
   VERSION              => 0,
   HTTPCONNECT          => 'tiny',
   CURL                 => 'curl',
   UNIVERSE             => undef, # Via Cmd line option 
   DBFILE               => undef, # Name of the output sqlite DB file 
   SQLITE               => "/usr/bin/sqlite3",
   GZIP                 => 0,
   DROPTABLES           => 1,
   TZOFFSET             => undef, # Set by unixtime_to_printable
   AUTORUN_SQLITE       => -t STDOUT , # If STDOUT is NOT redirected, we automatically run sqlite3
   STATUS_MSG_TO_STDERR => 0,
   FOLLOWER_LAG_MINIMUM => 1000, # Collect follower lag if GEQ this value 
   REGION               => {},    # Will be populated later 
   UNIV_DETAILS         => undef, # Will be populated later 
);

#---- Start ---
my ($YBA_API,$SQL_OUTPUT_FH, $db);
warn "-- ", $opt{STARTTIME_PRINTABLE}, " : Moses version $VERSION \@$opt{LOCALHOST} starting ...", "\n";
Initialize();

Get_and_Parse_tablets_from_tservers();

#---- Wrapup code -----
$db->putlog (TimeDelta("$db->{SQLOUTPUTRECORDCOUNT} SQL stmts generated"));

#$db->Insert_Post_Population_SQL();
$db->putlog(TimeDelta("Database Population completed in " . (time() -  $opt{STARTTIME}) . " sec."));
    # Note - tricky SQLITE date function call below - funny quoting required, to fool 'putlog'
$db->putlog("Database Population (SQL loading) completed (UTC)'||datetime('now')||'",4, q|strftime('%s','now')|);
$db->Create_Views();
$SQL_OUTPUT_FH and close ($SQL_OUTPUT_FH);

warn TimeDelta("COMPLETED. '$opt{DBFILE}' Created " , $opt{STARTTIME}),"\n";
$opt{DBFILE}=~/sqlite$/ and warn "\t RUN: sqlite3 -header -column $opt{DBFILE}\n";

exit 0;
#----------------------------------------------------------------------------------------------
sub Get_and_Parse_tablets_from_tservers{

  for my $n (@{ $opt{NODES} }){
      next unless $n->{isTserver};
      if ( $n->{state} ne  'Live'){
         warn "-- Node $n->{nodeName} $n->{Tserver_UUID} is $n->{state} .. skipping\n";
         next;
      }

      my $tabletCount = 0;
      print "SELECT '", TimeDelta("Processing tablets on $n->{nodeName} $n->{Tserver_UUID} (Idx $n->{nodeIdx})..."),"';\n";
      my $html_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/tablets?raw","BASE_URL_UNIVERSE",1); # RAW

      # Open the html text as a file . Read it using "</tr>\n" as the line ending, to get one <tr>(tablet) at a time 
      open my $f,"<",\$html_raw or die $!;

      local $/="</tr>\n";
      my $row=0;
	  my %leaders;
      my $header =<$f>;
      $header or die "ERROR: Cant read header from node/tablet  HTML";
      #print "HDR: $header";
      my @fields = map{tr/ -/_/;uc } $header=~m{<th>([^<]+)</th>}sg;
      $db->CreateTable("TABLET","node_uuid", 
                       map {$Tablet::is_numeric{$_}? "$_ INTEGER":$_}
                         @Tablet::db_fields);
      $db->putsql("BEGIN TRANSACTION; --Tablets for tserver $n->{nodeName}");

      Tablet::SetFieldNames(@fields);
      
      while (<$f>){
          next unless m/<td>/;
          my $t = Tablet::->new_from_tr($_);
          $tabletCount++;
          $leaders{$t->{LEADER}} ++;
          $db->Insert_Tablet($t, $n->{Tserver_UUID});
      }
      close $f;
      Collect_Follower_Lag_metrics($n);
      $db->putsql("END TRANSACTION; --Tablets for  tserver $n->{nodeName}");
	  $db->putlog("Found $tabletCount tablets on $n->{nodeName}:"
	       . join (", ",map{ " $leaders{$_} leaders  on $_" } sort keys %leaders));
      print "SELECT '",TimeDelta("Found $tabletCount tablets on $n->{nodeName}:"
         . join (", ",map{ " $leaders{$_} leaders  on $_" } sort keys %leaders)),"';\n";
  }
  
  $db->putsql("CREATE UNIQUE INDEX tablet_idx ON tablet (node_uuid,tablet_uuid);");
}
#----------------------------------------------------------------------------------------------
sub Collect_Follower_Lag_metrics{
   my ($n) = @_;
   my $lags = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/metrics?metrics=follower_lag_ms","BASE_URL_UNIVERSE");
   my $ts = time();

	 #$db->putsql($ts,"TABLETMETRIC","NODE:$n->{nodeUuid}\nmetric:follower_lag_ms"
	#	              ,$YBA_API->{json_string},"text/json");
  my $bj = JSON::Tiny::decode_json($YBA_API->{json_string});
   # Put stuff into tablemetrics table 
	for my $metricInfo (@$bj){
	   for my $m( @{$metricInfo->{metrics}} ){
      next unless $m->{value} >= $opt{FOLLOWER_LAG_MINIMUM};
	     $db->putsql("INSERT INTO TABLETMETRIC VALUES("
	           #timestamp INTEGER, node-uuid , tablet_id TEXT, metric_name TEXT, metric_value NUMERIC
                  . $ts
                  . qq|,"$n->{Tserver_UUID}"|  # Node UUID 
                  . qq|,"$metricInfo->{id}"| # Tablet ID
                  . qq|,"$m->{name}"| 
                  . qq|,| . $m->{value}
                  . ");");
	   }
	}

}
#----------------------------------------------------------------------------------------------
sub Initialize{

    GetOptions (\%opt, qw[DEBUG! HELP! VERSION!
                        API_TOKEN=s YBA_HOST=s UNIVERSE=s
                        GZIP! DBFILE=s SQLITE=s GZIP! DROPTABLES!
                        HTTPCONNECT=s CURL=s FOLLOWER_LAG_MINIMUM=i]
               ) or die "ERROR: Invalid command line option(s). Try --help.";
    if ($opt{DEBUG}){
      print "-- ","DEBUG: Option $_\t="
          .(defined $opt{$_} ? ref $opt{$_} eq "ARRAY"? join (",",@{$opt{$_}}): $opt{$_}
                             : "*Not Defined*")
          .";\n" 
          for sort keys %opt;
    }
    if ($opt{HELP}){
      warn $HELP_TEXT;
      exit 0;
    }
    $opt{VERSION} and exit 1; # Version request already fulfilled.
    # Initialize connection to YBA 
    $YBA_API = Web::Interface::->new();
   eval { $opt{YBA_JSON} = $YBA_API->Get("/customers","BASE_URL_API_V1") };
   if ($@  or  (ref $opt{YBA_JSON} eq "HASH" and $opt{YBA_JSON}{error})){
     die "ERROR:Unable to `get` YBA API customer info - Bad API_TOKEN?:$@"; 
   }
   ## All is well - we got the info in $opt{YBA_JSON}
   $opt{CUST_UUID} = $opt{YBA_JSON}[0]{uuid};
   $YBA_API->Set_Value("CUST_UUID", $opt{YBA_JSON}[0]{uuid});
   $opt{DEBUG} and print "--DEBUG: Customer $opt{YBA_JSON}[0]{name} = $opt{YBA_JSON}[0]{uuid}\n";
   
   $opt{UNIVERSE_LIST} = $YBA_API->Get("/customers/$YBA_API->{CUST_UUID}/universes","BASE_URL_API_V1");
   for my $u (@{$opt{UNIVERSE_LIST}}){
      $opt{DEBUG} and print "--DEBUG: Universe: $u->{name}\t $u->{universeUUID}\n";
      if ($opt{UNIVERSE}  and  $u->{name} =~/$opt{UNIVERSE}/i){
         print "-- Selected Universe $u->{name}\t $u->{universeUUID}\n";
		 $opt{STATUS_MSG_TO_STDERR} and warn "-- Selected Universe $u->{name}\t $u->{universeUUID}\n";
         $YBA_API->Set_Value("UNIV_UUID",$u->{universeUUID});
         last;
      }
   }
   if (! $YBA_API->{"UNIV_UUID"}){
       warn "Please select a universe name (or unique part thereof) from:\n";
       warn "\t$_->{name}\n"  for (@{$opt{UNIVERSE_LIST}});
       die "ERROR: --UNIVERSE ($opt{UNIVERSE}) incorrect or unspecified\n";
   }
   # -- Universe  Node details -
   $opt{UNIV_DETAILS} =  $YBA_API->Get(""); # Huge Univ JSON 
   $opt{DEBUG} and print "--DEBUG:UNIV: $_\t","=>",$opt{UNIV_DETAILS}{$_},"\n" for qw|name creationDate universeUUID version |;
  #my ($universe_name) =  $json_string =~m/,"name":"([^"]+)"/;
  if ($opt{UNIV_DETAILS}{name}){
     print TimeDelta(join("", "UNIVERSE: ", $opt{UNIV_DETAILS}{name}," on ", $opt{UNIV_DETAILS}{universeDetails}{clusters}[0]{userIntent}{providerType},
           " ver ",$opt{UNIV_DETAILS}{universeDetails}{clusters}[0]{userIntent}{ybSoftwareVersion})),"\n";
  }else{
     die "ERROR: Universe info not found \n";
  }
  $opt{NODES} = Extract_nodes_From_Universe($opt{UNIV_DETAILS});
  # Find Master/Leader 
  $opt{MASTER_LEADER}      = $YBA_API->Get("/leader")->{privateIP};
  $opt{DEBUG} and print "--DEBUG:Master/Leader JSON:",$YBA_API->{json_string},". IP is ",$opt{MASTER_LEADER},".\n";
  my ($ml_node) = grep {$_->{private_ip} eq $opt{MASTER_LEADER}} @{ $opt{NODES} } or die "ERROR : No Master/Leader NODE found for $opt{MASTER_LEADER}";
  my $master_http_port = $opt{UNIV_DETAILS}{universeDetails}{communicationPorts}{masterHttpPort} or die "ERROR: Master HTTP port not found in univ JSON";
  
  #--- Initialize SQL output -----
  Setup_Output_Processing(); # Figure out if we are piping to sqlite etc.. 

  $db = DatabaseClass::->new(OUTPUTOBJ=>undef, DROPTABLES=>$opt{DROPTABLES});
  
  # Put Univ and node info into DB
  $db->Insert_nodes($opt{NODES});
  $db->CreateTable("gflags",qw|type key  value|);
  Extract_gflags_and_Region_info($opt{UNIV_DETAILS});
  
  # Extract Region info from nodes and store it
  
  # Get dump_entities JSON from MASTER_LEADER
  $opt{DEBUG} and print TimeDelta("DEBUG:Getting Dump Entities..."),"\n";  
  my $entities = $YBA_API->Get("/proxy/$ml_node->{private_ip}:$master_http_port/dump-entities","BASE_URL_UNIVERSE");
  # Analyze & save DUMP ENTITIES contained in  $YBA_API->{json_string} 
  Handle_ENTITIES_Data($entities);

  $db->CreateTable("tabletmetric","timestamp INTEGER","node_uuid TEXT","tablet_id TEXT",
                   "metric_name TEXT","metric_value NUMERIC");
  $db->CreateTable("metrics",qw|name node_uuid tablet_uuid|,"value NUMERIC"); # non-tablet metrics 
  Get_Node_Metrics();
}
#----------------------------------------------------------------------------------------------
sub Setup_Output_Processing{
	if (not $opt{AUTORUN_SQLITE}){
		$opt{DBFILE} = "STDOUT";
		$opt{STATUS_MSG_TO_STDERR} = 1; 
		return; # No output processing needed-this has been setup manually
	}
	if (! $opt{GZIP}){
		my $SQLITE_ERROR  = (qx|$opt{SQLITE} -version|=~m/([^\s]+)/  ?  0 : "Could not run SQLITE3: $!"); # Checks if sqlite3 can run
		if ($SQLITE_ERROR){
			warn "WARNING: $SQLITE_ERROR\n\t Creating compressed SQL (not a sqlite database)";
			$opt{GZIP} = 1;
		}
	}
    $opt{DBFILE} ||= join(".", unixtime_to_printable($opt{STARTTIME},"YYYY-MM-DD"),$opt{LOCALHOST},"tabletInfo",
                                $opt{UNIV_DETAILS}{name}, $opt{GZIP}?"sql.gz":'sqlite');
	my $output_sqlite_dbfilename = $opt{DBFILE};
	if (-e $output_sqlite_dbfilename){
		my $mtime     = (stat  $output_sqlite_dbfilename)[9];
		my $rename_to = $output_sqlite_dbfilename .".". unixtime_to_printable($mtime,"YYYY-MM-DD-HH-MM") ;
		if  (-e $rename_to){
			die "ERROR:Files $output_sqlite_dbfilename and  $rename_to already exist. Please cleanup!";
		} 
		warn "WARNING: Renaming Existing file $output_sqlite_dbfilename to $rename_to.\n";
		rename $output_sqlite_dbfilename, $rename_to or die "ERROR:cannot rename: $!";
		sleep 2; # Allow time to read the message 
	}
	if ($opt{GZIP}){
		$opt{STATUS_MSG_TO_STDERR} = 1;
		open ($SQL_OUTPUT_FH, "|-", "gzip -c > $output_sqlite_dbfilename")
		     or die "ERROR: Could not start gzip : $!";
	}else{
		open ($SQL_OUTPUT_FH, "|-", "$opt{SQLITE} $output_sqlite_dbfilename")
			or die "ERROR: Could not start sqlite3 : $!";
	}
	select $SQL_OUTPUT_FH; # All subsequent "print" goes to this file handle.
}
#----------------------------------------------------------------------------------------------
sub Handle_ENTITIES_Data{
	my ($bj) = @_; # Entities decoded JSON 
	$opt{DEBUG} and print "--DEBUG:IN: Handle_ENTITIES_Data\n";
    $db->CreateTable("keyspaces","id TEXT PRIMARY KEY","name TEXT", "type TEXT"); # -- YCQL
    $db->CreateTable("tables","id TEXT PRIMARY KEY",qw|keyspace keyspace_id name state uuid tableTyp 
                relationType |,"sizeBytes NUMERIC", "walSizeBytes NUMERIC", "isIndexTable INTEGER","pgSchemaName TEXT","ttlInSeconds INTEGER");
    $db->CreateTable("tablecol","tableid TEXT", "isPartitionKey INTEGER","isClusteringKey INTEGER",qw| columnOrder  sortOrder  
                                    name  type partitionKey  clusteringKey|);
    $db->CreateTable("ent_tablets",qw|id  table_id state type server_uuid addr leader |); #-- Multiple tablet replicas w same ID
    $db->CreateTable("namespaces",qw|namespaceUUID  name  tableType|); #-- YSQL 

	# We get a giant JSON dump of entities .. parse it 
	#{"keyspaces":[{"keyspace_id":"..","keyspace_name":"system","keyspace_type":"ycql"},
	# {"keyspace_id":"7c51fb494aaf4da786c5ffd4175f4f3c","keyspace_name":"vijay","keyspace_type":"ycql"}],"tables":[{"table_id":"000...
	#tablets":[{"table_id":"sys.catalog.uuid","tablet_id":"00000000000000000000000000000000","state":"RUNNING"},{"table_id":"000033e80000300080000000000042e3","tablet_id":"003353a4627048fb8a9733f353ccf903","state":"RUNNING","replicas":[{"type":"VOTER","server_uuid":"92b2779d3a5f496fb0ad7b846f1270e4","addr":"10.231.0.66:9100"},{..],"leader":"92b2779d3a5f496fb0ad7b846f1270e4"},
	#my $bj = JSON::Tiny::decode_json($body);
    $db->putsql("BEGIN TRANSACTION; -- Entities");
    for my $ks (@{ $bj->{keyspaces} }){
	 	# Add to KEYSPACES table #$opt{DEBUG} and print "--DEBUG: Keyspace $ks->{keyspace_name} ($ks->{keyspace_id}) type $ks->{keyspace_type}\n";
		$ks->{keyspace_id} =~tr/-//d;
		$db->putsql("INSERT INTO keyspaces VALUES('" 
		             .join("','", $ks->{keyspace_id},$ks->{keyspace_name},$ks->{keyspace_type})
			         ."');");
	}
    for my $t (@{ $bj->{tables} }){
		$db->putsql( "INSERT INTO tables (id,keyspace_id,name,state) VALUES('"
		       . join("','", $t->{table_id}, $t->{keyspace_id},  $t->{table_name}, $t->{state})
			   . "');");
	}
	
	my %node_by_ip;
    for my $t (@{ $bj->{tablets} }){
	 	my $replicas = $t->{replicas} ; # AOH
	 	my $l        = $t->{leader} || "";
		for my $r (@$replicas){
		   $db->putsql( "INSERT INTO ent_tablets VALUES('"
		       . join("','", $t->{tablet_id}, $t->{table_id}, $t->{state}, $r->{type}, $r->{server_uuid},$r->{addr},$l )
			   . "');");
           my ($node_ip) = $r->{addr} =~/([\d\.]+)/ or next;
           next if $node_by_ip{$node_ip}; # Already setup 
           $node_by_ip{$node_ip} = $r->{server_uuid}; # Tserver UUID 
		}
	}
	$opt{DEBUG} and printf "--DEBUG: %d Keyspaces, %d tables, %d tablets\n", 
	                     scalar(@{ $bj->{keyspaces} }),scalar(@{ $bj->{tables} }), scalar(@{ $bj->{tablets} });
    
    # Fixup Node UUIDs : The ones in the Universe JSON are useless - so we update from tablets with TSERVER uuid 
    for my $n (@{ $opt{NODES} }){
       $n->{Tserver_UUID} = $node_by_ip{$n->{private_ip}}; # update in-mem info
    }
    $db->putsql( "UPDATE NODE "
               . "SET nodeUuid=(select server_uuid FROM ent_tablets "
               . "WHERE  substr(addr,1,instr(addr,\":\")-1) = private_ip limit 1);\n");
    $db->putsql("END TRANSACTION; -- Entities");
}
#----------------------------------------------------------------------------------------------
sub Extract_nodes_From_Universe{
    my ($univ_hash, $callback) = @_;
	
	my @node;
	my $count=0;
	for my $n (@{  $univ_hash->{universeDetails}{nodeDetailsSet} }){
       push @node, my $thisnode = {map({$_=>$n->{$_}||''} qw|nodeIdx nodeName nodeUuid azUuid isMaster
                                  	   isTserver ysqlServerHttpPort yqlServerHttpPort state tserverHttpPort 
									   tserverRpcPort masterHttpPort masterRpcPort nodeExporterPort|),
	                              map({$_=>$n->{cloudInfo}{$_}} qw|private_ip public_ip az region |) };
       $thisnode->{$_} =~tr/-//d for grep {/uuid/i} keys %$thisnode;
       $callback and $callback->($thisnode, $count);
       $count++;
    }
    return [@node];	
}
#------------------------------------------------------------------------------------------------
sub Extract_gflags_and_Region_info{
	my ($univ_hash) = @_;
	for my $k (qw|universeName provider providerType replicationFactor numNodes ybSoftwareVersion enableYCQL
             	enableYSQL enableYEDIS nodePrefix |){
	   next unless defined ( my $v= $univ_hash->{universeDetails}{clusters}[0]{userIntent}{$k} );
	   $db->putsql("INSERT INTO gflags VALUES ('CLUSTER','$k','$v');");
	}
	for my $flagtype (qw|masterGFlags tserverGFlags |){
	   next unless my $flag = $univ_hash->{universeDetails}{clusters}[0]{userIntent}{$flagtype};
	   for my $k(sort keys %$flag){
		  my $v = $flag->{$k}; 
	      $db->putsql("INSERT INTO gflags VALUES ('$flagtype','$k','$v');");
	   }
	}

	for my $region (@{ $univ_hash->{universeDetails} {clusters} [0]{placementInfo}{cloudList}[0]{regionList} }){
		my $preferred = 0;
		my $az_node_count = 0;
		for my $az ( @{ $region->{azList} } ){
			$az->{isAffinitized} and $preferred++;
			$az_node_count += $az->{numNodesInAZ};
		}
		$opt{REGION}{$region->{name}}{PREFERRED}     = $preferred;
		$opt{REGION}{$region->{name}}{UUID}          = $region->{uuid};
		$opt{REGION}{$region->{name}}{AZ_NODE_COUNT} = $az_node_count;
		$opt{DEBUG} and print "--DEBUG:REGION $region->{name}: PREFERRED=$preferred, $az_node_count nodes, $region->{uuid}.\n";
	}
}
#------------------------------------------------------------------------------------------------
sub Get_Node_Metrics{
  my %post_process;
  
  my %metric_handler=(
     ql_read_latency    => sub{my ($m,$table,$val)=$_[1]=~/^(\w+)\{table_id="(\w+).+?\s(\d+)/;
                               $post_process{$m}{$_[0]}{$table}=$val;},
     server_uptime_ms   => sub{my ($m,$val)=$_[1]=~/^(\w+).+?\s(\d+)/;save_metric($m,$_[0],0,$val)},
  );
  my $regex = "^(" . join("|^",keys(%metric_handler)). ")";
  $regex = qr|$regex|;
  $db->putsql("BEGIN TRANSACTION; -- Node Metrics");

  for my $n (@{ $opt{NODES} }){
    next unless $n->{state} eq  'Live';
    if ($n->{isTserver}){
       my $metrics_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/prometheus-metrics?reset_histograms=false",
                                       "BASE_URL_UNIVERSE",1); # RAW
       Read_this_buffer_w_callback(\$metrics_raw,
         sub{
            my ($metric) = $_[0]=~$regex or return;
            $metric_handler{$metric} -> ($n->{Tserver_UUID},$_[0]);
         }
       );
    }
    if ($n->{isMaster}){
       my $metrics_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{masterHttpPort}/prometheus-metrics?reset_histograms=false",
                                       "BASE_URL_UNIVERSE",1); # RAW
       Read_this_buffer_w_callback(\$metrics_raw,
         sub{
            my ($metric) = $_[0]=~$regex or return;
            $metric_handler{$metric} -> ($n->{nodeUuid},$_[0]); # Note - we do not have master-UUID 
         }
       );
    }
  }
  
  for my $sum_key (grep {m/_sum$/} keys(%post_process)){
     my ($metric_base_name) = $sum_key=~m/^(\w+)_sum$/; # get the base 
     my $count_metric_name = "${metric_base_name}_count";
     my $count_metric = $post_process{$count_metric_name}  or next;
     for my $node_uuid (keys %{ $post_process{$sum_key} }){
        next unless $count_metric->{$node_uuid};
        for my $table_uuid (keys %{$count_metric->{$node_uuid}}){
          my $count   = $count_metric->{$node_uuid}{$table_uuid} or next;
          my $avg_val = $post_process{$sum_key}{$node_uuid}{$table_uuid} / $count;
          save_metric($metric_base_name."_avg", $node_uuid, $table_uuid,sprintf('%.2f',$avg_val));
        } 
     }
  }

  $db->putsql("END TRANSACTION; -- Node metrics");

}
#------------------------------------------------------------------------------------------------
sub save_metric{
  my ($metric,$node_uuid,$table_uuid,$value)=@_;
  $db->putsql("INSERT INTO METRICS VALUES('$metric','$node_uuid','$table_uuid',$value);");
}
#------------------------------------------------------------------------------------------------
sub Read_this_buffer_w_callback{
   my ($buf_ref, $callback, $delim) = @_;
   open my $f,"<",$buf_ref or die "Cannot open buffer as file:$!";
   $delim and local $/=$delim;
   while(<$f>){
      $callback->($_);
   }
   close $f;
}
#------------------------------------------------------------------------------------------------
sub unixtime_to_printable{
	my ($unixtime,$format, $showTZ) = @_;
	my ($sec,$min,$hour,$mday,$mon,$year,$wday,$yday,$isdst) = localtime($unixtime);
	$opt{TZOFFSET} ||= do{my $tz = (localtime time)[8] * 60 - POSIX::mktime(gmtime 0) / 60;
                                      	sprintf "%+03d:%02d", $tz / 60, abs($tz) % 60};
	if (not defined $format  or  $format eq "YYYY-MM-DD HH:MM:SS"){
       return sprintf("%04d-%02d-%02d %02d:%02d:%02d", $year+1900, $mon+1, $mday, $hour, $min, $sec)
	        . ($showTZ? " " . $opt{TZOFFSET} : "");
	}
	if ($format eq "YYYY-MM-DD HH:MM"){
       return sprintf("%04d-%02d-%02d %02d:%02d", $year+1900, $mon+1, $mday, $hour, $min)
	          . ($showTZ? " " . $opt{TZOFFSET} : "");
	}
	if ($format eq "YYYY-MM-DD"){
		  return sprintf("%04d-%02d-%02d", $year+1900, $mon+1, $mday)
		         . ($showTZ? " " . $opt{TZOFFSET} : "");
	}
	if ($format eq "HMS"){
		  return sprintf("%02d:%02d:%02d", $hour,$min,$sec)
		         . ($showTZ? " " . $opt{TZOFFSET} : "");
	}
	if ($format eq "YYYY-MM-DD-HH-MM"){
       return sprintf("%04d-%02d-%02d-%02d-%02d", $year+1900, $mon+1, $mday, $hour, $min)
	          . ($showTZ? " " . $opt{TZOFFSET} : "");
	}	
	die "ERROR: Unsupported format:'$format' ";
}
#----------------------------------------------------------------------------------------------
sub TimeDelta{
  my ($msg, $start_time) = @_;
  
  my $prev_time = $start_time || $opt{CURRENT_TIME};
  $opt{CURRENT_TIME} = time();
  my $delta = $opt{CURRENT_TIME} - $prev_time;

  my $returnmsg = "-- " . unixtime_to_printable($opt{CURRENT_TIME},"HMS") . " " . $msg;
  # The leading "--" is REQUIRED , because that makes this a SQL comment, and all output is SQL
  return $returnmsg if $delta < 61;

  $returnmsg .= " after " . sprintf("%d minutes %d seconds",$delta / 60, $delta % 60);
  $opt{STATUS_MSG_TO_STDERR} and warn "$msg\n";
  return   $returnmsg;
}

###############################################################################
############### C L A S S E S                             #####################
###############################################################################

######################################################################################
BEGIN{
package DatabaseClass;
use warnings;
use strict;
our $SCHEMA_VERSION = "1.0";

my $next_unique_number = int(rand(999)); # Starting at random

my %DBINFO =(
    LOG  => { FIELDS=>[qw|timestamp level message|],
       CREATE=> <<"     __LOG__",
       CREATE TABLE  IF NOT EXISTS LOG(
       id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
       timestamp INTEGER NOT NULL,
       level INTEGER,
       message TEXT
       );
     __LOG__
     DROPPABLE => 0,
    },
    GRIDXX => { FIELDS => [qw|ipaddr macaddr nodes hfscreatetime systemid timestamp HOSTNAME ISSOURCEGRID|],
      CREATE=> <<"     __GRID__",
       CREATE TABLE  IF NOT EXISTS xxxnode(
       id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
       ipaddr TEXT NOT NULL,
       macaddr TEXT,
       nodes INTEGER,
       hfscreatetime INTEGER NOT NULL,
       systemid TEXT,
       timestamp INTEGER,
       HOSTNAME TEXT,
       ISSOURCEGRID INTEGER
       );
     __GRID__
     DROPPABLE => 1,
    },TAABLE => {
       CREATE=><<"     __DOMAIN__",
       CREATE TABLE  IF NOT EXISTS xxxdomain(
       id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
       gridid INTEGER NOT NULL REFERENCES GRID(id) ON DELETE CASCADE,
       name     TEXT NOT NULL,
	   cid      TEXT,
       parentid INTEGER,
       CONTACTINFO TEXT
       );
       --CREATE UNIQUE INDEX IF NOT EXISTS by_domainid ON DOMAIN (id);  
       --CREATE UNIQUE INDEX IF NOT EXISTS by_gridid ON DOMAIN (gridid,id); 
       --CREATE UNIQUE INDEX IF NOT EXISTS by_domain_name ON DOMAIN (name,gridid);  
     __DOMAIN__
     DROPPABLE => 1,
    }, 
    
);


sub new {
    my ($class,%att) = @_;
    $opt{DEBUG} and print "---DEBUG: Creating NEW DatabaseClass object\n";
    $class = ref $class if ref $class;

    my $self = bless \%att , $class;

    $self->putsql("-- Generating $opt{DBFILE} by $0 $VERSION on $opt{LOCALHOST} by " . ($ENV{USER}||$ENV{USERNAME}) . " on " . scalar(localtime(time)));
    $self->putsql("PRAGMA SCHEMA_VERSION=$SCHEMA_VERSION;");
    $self->putsql("PRAGMA foreign_keys = ON;");
    $self->putsql("PRAGMA recursive_triggers = TRUE;");
    $opt{DEBUG} and sleep 3; # Allow for debugger break 
    for my $table(sort keys %DBINFO){ # Note: $table is UPPER-CASE only(in data)
      if ($self->{DROPTABLES}){
          $opt{DEBUG} and print "--DEBUG: Dropping indexes for $table.\n";
          my @index_names = $DBINFO{$table}{CREATE} =~/CREATE\s+\w*\s*INDEX .+\s(\w+)\s+ON\s+\w+\s+\(/ig;
          $opt{DEBUG} and print "--DEBUG: Dropping @index_names\n";  
          $self->putsql("DROP INDEX IF EXISTS $_;") for @index_names;
          $self->putsql("DROP TABLE IF EXISTS $table;") if $DBINFO{$table}{DROPPABLE};

          my $field_count = 999; # Prevent this from  looping
          while ($DBINFO{$table}{CREATE}=~/(\w+)\s+(\w+).*$/mg){
               if (uc($2) eq "TEXT"){
                  $self->{VALUE_QUOTER}{$table}{uc $1} = sub{defined $_[0] ? return "'$_[0]'" : return 'NULL'};
                  $opt{DEBUG} and print "-- Table \[$table] field ($1) created TEXT sub ",uc($1),"\n";
               }elsif (uc($2) eq "INTEGER"){
                  $self->{VALUE_QUOTER}{$table}{uc $1} = sub{defined $_[0] ? return $_[0] : return 'NULL'};
                  $opt{DEBUG} and print "-- Table \[$table] field ($1) created INTEGER sub ",uc($1),"\n";
               }
               last if $field_count-- < 0;
          } 
      }
      $self->putsql ($DBINFO{$table}{CREATE});
    }
    ##$self->putsql(DB_Views());
    my $t = time();
    $self->putlog($_,7,$t) for (
         "--STARTUP --",
         "PROG_VERSION $VERSION",
         "SCHEMA_VERSION $SCHEMA_VERSION",
         "LOCALHOST $opt{LOCALHOST}",
         "USER " .($ENV{USER}||$ENV{USERNAME}),
         "DATE " .  main::unixtime_to_printable(time())
         );
    for (sort keys %opt){
        my $val = defined ($opt{$_})? $opt{$_} : "*Not Defined*";
        ref $val eq "ARRAY" and $val = join(",", @$val);
        $_=~/\bpassword|AP\b/i and $val="*****";
        $self->putlog("PROG_OPTION $_=$val",7,$t) ;
    }
    # Note - tricky SQLITE date function call below - funny quoting required, to fool 'putlog'
    $self->putlog("Database Population start '||datetime('now')||'",4, q|strftime('%s','now')|);
    return $self;
  }
  
###############################################################################
sub putsql{
  my ($self,$txt) = @_;
  defined $txt or return;
  #print $sqlf "$txt\n";
  print $txt,"\n" ;
  $self->{SQLOUTPUTRECORDCOUNT}++;
}
###############################################################################
sub putlog{
  my ($self, $txt,$level, $unixtime) = @_;
    defined $txt or return;
  $level ||=7; # Info
  $unixtime ||= time();
  $self->putsql("INSERT INTO LOG (timestamp,level,message) VALUES("
       . $unixtime .",$level,'$txt');" );
}

sub CreateTable{
  my ($self,$name,@fields) = @_;
  $self->putsql("CREATE TABLE IF NOT EXISTS $name (" .
        join(",",@fields) . ");");
}
sub Insert_nodes{
  my ($self,$nodes) = @_;
  $self->CreateTable("NODE", sort keys %{$nodes->[0]} );
  $self->putsql("-- NOTE: nodeUUID value is later updated to be TSERVER_UUID");
  for my $n(@$nodes){
    $self->putsql("INSERT INTO NODE VALUES('" .
                 join("','", map{$n->{$_}} sort keys %$n ). "');");
  }
}

sub Insert_Tablet{
  my ($self,$t,$node_uuid) = @_;
  $self->putsql("INSERT into TABLET VALUES('$node_uuid'," . 
                $t->Get_csv_quoted_values() . ");");
}

sub Create_Views{
  my ($self) = @_;
  $self->putsql(<<"__SQL__");
  CREATE VIEW version_info AS 
    SELECT '$0' as program, '$VERSION' as version, '$opt{STARTTIME_PRINTABLE}' AS run_on, '$opt{LOCALHOST}' as host;

  CREATE VIEW tablets_per_node AS 
    SELECT node_uuid,min(public_ip) as node_ip,min(region ) as region,  count(*) as tablet_count,
           sum(CASE WHEN private_ip = leader THEN 1 ELSE 0 END) as leaders,
           count(DISTINCT table_name) as table_count
	FROM tablet,node 
	WHERE isTserver  and node.nodeuuid=node_uuid 
	GROUP BY node_uuid
	ORDER BY tablet_count;

  CREATE VIEW tablet_replica_detail AS
    SELECT t.namespace,t.table_name,t.table_uuid,t.tablet_uuid,
	count(DISTINCT LEADER) as leader_count, count(*) as replicas
	from tablet t
	GROUP BY t.namespace,t.table_name,t.table_uuid,t.tablet_uuid;

  CREATE VIEW tablet_replica_summary AS
     SELECT replicas,count(*) as tablet_count FROM  tablet_replica_detail GROUP BY replicas;

  CREATE VIEW leaderless AS 
     SELECT t.tablet_uuid, replicas,t.namespace,t.table_name,node_uuid,private_ip ,leader_count
	 from tablet t,node ,tablet_replica_detail trd
	 WHERE  node.isTserver  AND nodeuuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid  
		   AND trd.leader_count !=1;

  CREATE VIEW delete_leaderless_be_careful AS 
    SELECT '\$HOME/tserver/bin/yb-ts-cli delete_tablet '|| tablet_uuid ||' -certs_dir_name \$TLSDIR -server_address '||private_ip ||':9100  \$REASON_tktnbr'
	   AS generated_delete_command
     FROM leaderless;

--  Based on  yb-ts-cli unsafe_config_change <tablet_id> <peer1> (undocumented)
--  https://phorge.dev.yugabyte.com/D12312
  CREATE VIEW UNSAFE_Leader_create AS
        SELECT  '\$HOME/tserver/bin/yb-ts-cli --server_address='|| private_ip ||':'||tserverrpcport 
        || ' unsafe_config_change ' || t.tablet_uuid
		|| ' ' || node_uuid
		|| ' -certs_dir_name \$TLSDIR;sleep 30;' AS cmd_to_run
	 from tablet t,node ,tablet_replica_detail trd
	 WHERE  node.isTserver  AND nodeuuid=node_uuid
	       AND  t.tablet_uuid=trd.tablet_uuid  
		   AND trd.leader_count !=1;
__SQL__
}
1;  
} # End of package DatabaseClass
###############################################################################
##########################################################################################################
BEGIN{
package Tablet;
my (@fields); 
our @db_fields =  qw|TABLET_UUID NAMESPACE TABLE_NAME TABLE_UUID STATE HIDDEN LEADER FOLLOWERS 
                    NUM_SST_FILES PARTITION LAST_STATUS
                    SST_FILES SST_FILES_UNCOMPRESSED TOTAL WAL_FILES |;
our %is_numeric = map{$_=>1} qw|NUM_SST_FILES SST_FILES SST_FILES_UNCOMPRESSED TOTAL WAL_FILES|;
   # <tr><td>yugabyte</td>
   #    <td>account_balance_secondary_index</td><td>000033e80000300080000000000042e3</td>
   #    <td><a href="/tablet?id=003353a4627048fb8a9733f353ccf903">003353a4627048fb8a9733f353ccf903</a></td>
   #     <td>hash_split: [0x5FFD, 0x7FFC)</td>
   #      <td>RUNNING</td><td>false</td>
   #      <td>0</td>
   #       <td><ul>
   #           <li>Total: 2.00M<li>Consensus Metadata: 1.5K
   #           <li>WAL Files: 2.00M<li>SST Files: 0B
   #           <li>SST Files Uncompressed: 0B
   #        </ul></td>
   #      <td><ul>
   #          <li>FOLLOWER: 10.231.0.64</li>
   #          <li>FOLLOWER: 10.231.0.65</li>
   #          <li><b>LEADER: 10.231.0.66</b></li>
   #      </ul>
   #      </td>
   #      <td>account_balance_secondary_index0</td></tr>


my %kilo_multiplier=(
		BYTES	=> 1,
        B       => 1,
		KB		=> 1024,
        K       => 1024,
		MB		=> 1024*1024,
        M       => 1024*1024,
		GB		=> 1024*1024*1024,
        G       => 1024*1024*1024,
	);

sub new_from_tr{
    my ($class, $line) = @_;
    # This parses a <tr> block for one tablet, extracts fields and returns a tablet object
    my $h=0;
    my %val = map{$fields[$h++] => $_} $line=~m{<td>(.+?)</td>}gs;
    
    ($val{TABLET_UUID}) = $val{TABLET_ID} =~m/>(\w+)</;
    $val{FOLLOWERS} = join ",", $val{'RAFTCONFIG'} =~m/FOLLOWER: ([^<]+)/g ; # a CSV string 
    ($val{LEADER})    =  $val{'RAFTCONFIG'} =~m/LEADER: ([^<]+)/;
    my %disk = $val{ON_DISK_SIZE} =~m/>(\w.+?): ([^<]+)/g;

    $val{do{tr/ -/__/;uc }} = $disk{$_} for keys %disk;
        # Convert disk values to numeric (xlate K, B etc)
    for my $k (qw|SST_FILES SST_FILES_UNCOMPRESSED TOTAL WAL_FILES|){
        my ($numeric,$unit) = $val{$k} =~m/([\-\.\d]+)(\w+)/;
        $val{$k} = $numeric *  ($kilo_multiplier{ uc $unit } || 1);
    }    
    delete $val{$_} for qw|TABLET_ID RAFTCONFIG ON_DISK_SIZE|; # rm hashes - this object is now FLAT.
    return bless \%val, $class;
}

sub Print{
   my ($self) = @_;
   my $count=0;
   print "---- ";
   for my $f (@fields){
       if ($count++ > 5){
           $count=0;
           print "\n";
       }
       print "  $f=";
       my $v = $self->{$f};

       if ("ARRAY" eq ref ($v)){
         print "\[",join(", ",@$v),"]\n";
         $count = 0;
       }elsif ("HASH" eq ref($v)){
         print "{";
         for my $item(sort keys %$v){
           if (ref($v->{$item}) eq "ARRAY"){
              print " $item=\[",join(",", @{$v->{$item}}),"\] ";
              next;
           } 
           print "$item='$v->{$item}', ";
         }
         print "}\n";
         $count = 0;
       }else{
         print $v,";";
       }
   }
   print "\n";
}

sub SetFieldNames{ # Class method 
  my (@names_raw, $db) = @_;
  @fields = map {tr/ //d; uc $_} @names_raw;
}

sub Get_csv_quoted_values{
   my ($self) = @_;
   return join (",", map {$is_numeric{$_} ? $self->{$_}||0 : "'$self->{$_}'"} @db_fields);
}
1;
} # End of Tablet
##########################################################################################################
#==============================================================================
BEGIN{
package Web::Interface; # Handles communication with YBA API 

sub new{
    my ($class) = @_;
    for(qw|API_TOKEN YBA_HOST |){
        $opt{$_} or die "ERROR: Required parameter --$_ was not specified.\n";
    }
    for(qw|API_TOKEN |){
        (my $value=$opt{$_})=~tr/-_//d; # Extract and zap dashes
        my $len = length($value);
        next if $len == 32; # Expecting these to be exactly 32 bytes 
        warn "WARNING: Expecting 32 valid bytes in Option $_=$opt{$_} but found $len bytes. \n";
        sleep 2;
    }   
    my $self =bless {map {$_ => $opt{$_}||""} qw|HTTPCONNECT UNIV_UUID API_TOKEN YBA_HOST CUST_UUID| }, $class;
if ( $opt{YBA_HOST} =~m{^(http\w?://)}i ){
       $self->{HTTP_PREFIX} ="$1";
       $self->{YBA_HOST} = substr($self->{YBA_HOST},length($1));
    }else{
       $self->{HTTP_PREFIX} ="http://";
    }
    $self->Set_Value();
    $self->{HTTP_PREFIX} ||= substr($opt{YBA_HOST},0,5); # HTTP: or HTTPS
    if ($self->{HTTPCONNECT} eq "curl"){
          $self->{curl_base_cmd} = join " ", $opt{CURL}, 
                     qq|-kfs --request GET --header 'Content-Type: application/json'|,
                     qq|--header "X-AUTH-YW-API-TOKEN: $opt{API_TOKEN}"|;
          if ($opt{DEBUG}){
             print "--DEBUG:CURL base CMD: $self->{curl_base_cmd}\n";
          }
          return $self;
    }
    if ($self->{HTTP_PREFIX}=~/^https/i){
       my ($ok, $why) = eval { HTTP::Tiny->can_ssl() };
       if ($@ or not $ok){
             print "--WARNING: HTTP::Tiny does not support SSL.  Switching to curl.\n:";
             $opt{HTTPCONNECT} = "curl";
             return $class->new(); # recurse
       }
    }
    $self->{HT} = HTTP::Tiny->new( default_headers => {
                         'X-AUTH-YW-API-TOKEN' => $opt{API_TOKEN},
                         'Content-Type'      => 'application/json',
                         # 'max_size'        => 5*1024*1024, # 5MB 
                      });

    return $self;
}

sub Get{
    my ($self, $endpoint, $base, $raw) = @_;
    $self->{json_string}= "";
    my $url = $base ? $self->{$base} : $self->{BASE_URL_API_CUSTOMER};
    if ($self->{HTTPCONNECT} eq "curl"){
        $self->{json_string} = qx|$self->{curl_base_cmd} --url $url$endpoint|;
        if ($?){
           warn "ERROR: curl get '$endpoint' failed: $?\n";
           return {error=>$?};
        }
    }else{ # HTTP::Tiny
       $self->{raw_response} = $self->{HT}->get($url . $endpoint);
       if (not $self->{raw_response}->{success}){
          warn "ERROR: Get '$endpoint' failed with status=$self->{raw_response}->{status}: $self->{raw_response}->{reason}\n\tURL:$url$endpoint\n";
          $self->{raw_response}->{status} == 599 and warn "\t(599)Content:$self->{raw_response}{content};\n";
          return {error=> $self->{raw_response}->{status}};
       }
       $self->{json_string} = $self->{raw_response}{content};
    }
	if ($raw){
       return $self->{response} = $self->{json_string}; # Do not decode
    }
    $self->{response} = JSON::Tiny::decode_json( $self->{json_string} );
    return $self->{response};
}

sub Set_Value{
    my ($self,$k,$v) = @_;
    $k and $self->{$k} = $v;
    $self->{BASE_URL_API_CUSTOMER} = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/api/customers/$self->{CUST_UUID}/universes/$self->{UNIV_UUID}";
    $self->{BASE_URL_UNIVERSE}     = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/universes/$self->{UNIV_UUID}";
    $self->{BASE_URL_API_V1}       = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/api/v1";
}

} # End of  package Web::Interface;  
#==============================================================================
BEGIN{
package JSON::Tiny;

# Minimalistic JSON. Adapted from Mojo::JSON. (c)2012-2015 David Oswald
# License: Artistic 2.0 license.
# http://www.perlfoundation.org/artistic_license_2_0

use strict;
use warnings;
use Carp 'croak';
use Exporter 'import';
use Scalar::Util 'blessed';
use Encode ();
use B;

our $VERSION = '0.58';
our @EXPORT_OK = qw(decode_json encode_json false from_json j to_json true);

# Literal names
# Users may override Booleans with literal 0 or 1 if desired.
our($FALSE, $TRUE) = map { bless \(my $dummy = $_), 'JSON::Tiny::_Bool' } 0, 1;

# Escaped special character map with u2028 and u2029
my %ESCAPE = (
  '"'     => '"',
  '\\'    => '\\',
  '/'     => '/',
  'b'     => "\x08",
  'f'     => "\x0c",
  'n'     => "\x0a",
  'r'     => "\x0d",
  't'     => "\x09",
  'u2028' => "\x{2028}",
  'u2029' => "\x{2029}"
);
my %REVERSE = map { $ESCAPE{$_} => "\\$_" } keys %ESCAPE;

for(0x00 .. 0x1f) {
  my $packed = pack 'C', $_;
  $REVERSE{$packed} = sprintf '\u%.4X', $_ unless defined $REVERSE{$packed};
}

sub decode_json {
  my $err = _decode(\my $value, shift);
  return defined $err ? croak $err : $value;
}

sub encode_json { Encode::encode 'UTF-8', _encode_value(shift) }

sub false () {$FALSE}  ## no critic (prototypes)

sub from_json {
  my $err = _decode(\my $value, shift, 1);
  return defined $err ? croak $err : $value;
}

sub j {
  return encode_json $_[0] if ref $_[0] eq 'ARRAY' || ref $_[0] eq 'HASH';
  return decode_json $_[0];
}

sub to_json { _encode_value(shift) }

sub true () {$TRUE} ## no critic (prototypes)

sub _decode {
  my $valueref = shift;

  eval {

    # Missing input
    die "Missing or empty input\n" unless length( local $_ = shift );

    # UTF-8
    $_ = eval { Encode::decode('UTF-8', $_, 1) } unless shift;
    die "Input is not UTF-8 encoded\n" unless defined $_;

    # Value
    $$valueref = _decode_value();

    # Leftover data
    return m/\G[\x20\x09\x0a\x0d]*\z/gc || _throw('Unexpected data');
  } ? return undef : chomp $@;

  return $@;
}

sub _decode_array {
  my @array;
  until (m/\G[\x20\x09\x0a\x0d]*\]/gc) {

    # Value
    push @array, _decode_value();

    # Separator
    redo if m/\G[\x20\x09\x0a\x0d]*,/gc;

    # End
    last if m/\G[\x20\x09\x0a\x0d]*\]/gc;

    # Invalid character
    _throw('Expected comma or right square bracket while parsing array');
  }

  return \@array;
}

sub _decode_object {
  my %hash;
  until (m/\G[\x20\x09\x0a\x0d]*\}/gc) {

    # Quote
    m/\G[\x20\x09\x0a\x0d]*"/gc
      or _throw('Expected string while parsing object');

    # Key
    my $key = _decode_string();

    # Colon
    m/\G[\x20\x09\x0a\x0d]*:/gc
      or _throw('Expected colon while parsing object');

    # Value
    $hash{$key} = _decode_value();

    # Separator
    redo if m/\G[\x20\x09\x0a\x0d]*,/gc;

    # End
    last if m/\G[\x20\x09\x0a\x0d]*\}/gc;

    # Invalid character
    _throw('Expected comma or right curly bracket while parsing object');
  }

  return \%hash;
}

sub _decode_string {
  my $pos = pos;
  
  # Extract string with escaped characters
  m!\G((?:(?:[^\x00-\x1f\\"]|\\(?:["\\/bfnrt]|u[0-9a-fA-F]{4})){0,32766})*)!gc; # segfault on 5.8.x in t/20-mojo-json.t
  my $str = $1;

  # Invalid character
  unless (m/\G"/gc) {
    _throw('Unexpected character or invalid escape while parsing string')
      if m/\G[\x00-\x1f\\]/;
    _throw('Unterminated string');
  }

  # Unescape popular characters
  if (index($str, '\\u') < 0) {
    $str =~ s!\\(["\\/bfnrt])!$ESCAPE{$1}!gs;
    return $str;
  }

  # Unescape everything else
  my $buffer = '';
  while ($str =~ m/\G([^\\]*)\\(?:([^u])|u(.{4}))/gc) {
    $buffer .= $1;

    # Popular character
    if ($2) { $buffer .= $ESCAPE{$2} }

    # Escaped
    else {
      my $ord = hex $3;

      # Surrogate pair
      if (($ord & 0xf800) == 0xd800) {

        # High surrogate
        ($ord & 0xfc00) == 0xd800
          or pos($_) = $pos + pos($str), _throw('Missing high-surrogate');

        # Low surrogate
        $str =~ m/\G\\u([Dd][C-Fc-f]..)/gc
          or pos($_) = $pos + pos($str), _throw('Missing low-surrogate');

        $ord = 0x10000 + ($ord - 0xd800) * 0x400 + (hex($1) - 0xdc00);
      }

      # Character
      $buffer .= pack 'U', $ord;
    }
  }

  # The rest
  return $buffer . substr $str, pos $str, length $str;
}

sub _decode_value {

  # Leading whitespace
  m/\G[\x20\x09\x0a\x0d]*/gc;

  # String
  return _decode_string() if m/\G"/gc;

  # Object
  return _decode_object() if m/\G\{/gc;

  # Array
  return _decode_array() if m/\G\[/gc;

  # Number
  my ($i) = /\G([-]?(?:0|[1-9][0-9]*)(?:\.[0-9]*)?(?:[eE][+-]?[0-9]+)?)/gc;
  return 0 + $i if defined $i;

  # True
  return $TRUE if m/\Gtrue/gc;

  # False
  return $FALSE if m/\Gfalse/gc;

  # Null
  return undef if m/\Gnull/gc;  ## no critic (return)

  # Invalid character
  _throw('Expected string, array, object, number, boolean or null');
}

sub _encode_array {
  '[' . join(',', map { _encode_value($_) } @{$_[0]}) . ']';
}

sub _encode_object {
  my $object = shift;
  my @pairs = map { _encode_string($_) . ':' . _encode_value($object->{$_}) }
    sort keys %$object;
  return '{' . join(',', @pairs) . '}';
}

sub _encode_string {
  my $str = shift;
  $str =~ s!([\x00-\x1f\x{2028}\x{2029}\\"/])!$REVERSE{$1}!gs;
  return "\"$str\"";
}

sub _encode_value {
  my $value = shift;

  # Reference
  if (my $ref = ref $value) {

    # Object
    return _encode_object($value) if $ref eq 'HASH';

    # Array
    return _encode_array($value) if $ref eq 'ARRAY';

    # True or false
    return $$value ? 'true' : 'false' if $ref eq 'SCALAR';
    return $value  ? 'true' : 'false' if $ref eq 'JSON::Tiny::_Bool';

    # Blessed reference with TO_JSON method
    if (blessed $value && (my $sub = $value->can('TO_JSON'))) {
      return _encode_value($value->$sub);
    }
  }

  # Null
  return 'null' unless defined $value;


  # Number (bitwise operators change behavior based on the internal value type)

  return $value
    if B::svref_2object(\$value)->FLAGS & (B::SVp_IOK | B::SVp_NOK)
    # filter out "upgraded" strings whose numeric form doesn't strictly match
    && 0 + $value eq $value
    # filter out inf and nan
    && $value * 0 == 0;

  # String
  return _encode_string($value);
}

sub _throw {

  # Leading whitespace
  m/\G[\x20\x09\x0a\x0d]*/gc;

  # Context
  my $context = 'Malformed JSON: ' . shift;
  if (m/\G\z/gc) { $context .= ' before end of data' }
  else {
    my @lines = split "\n", substr($_, 0, pos);
    $context .= ' at line ' . @lines . ', offset ' . length(pop @lines || '');
  }

  die "$context\n";
}

# Emulate boolean type
package JSON::Tiny::_Bool;
use overload '""' => sub { ${$_[0]} }, fallback => 1;
1;	

}; #End of JSON::Tiny
#==============================================================================
#==============================================================================
#======================================================================