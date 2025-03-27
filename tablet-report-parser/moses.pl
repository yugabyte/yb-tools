#!/usr/bin/perl

our $VERSION = "0.37";
my $HELP_TEXT = << "__HELPTEXT__";
    It's a me, \x1b[1;33;100mmoses.pl\x1b[0m  Version $VERSION
               ========
 Get and analyze info on all tablets , config(gflags/xCluster) in the system.
 By default, output will be piped to sqlite3 to create a sqlite3 database,
 and run default reports.
 Moses also collects a snapshot of metrics (tables, tablet, xCluster lag).
 
\x1b[1;33;100mRun options:\x1b[0m
   --YBA_HOST         [=] <YBA hostname or IP> (Required) "[http(s)://]<hostname-or-ip>[:<port>]"
   --API_TOKEN        [=] <API access token>   (Required)
   --UNIVERSE         [=] <Universe-Name-or-uuid>  (Required(*). Name Can be partial, sufficient to be Unique)
   --CUSTOMER         [=] <Customer-uuid-or-name> (Optional. Required if more than one customer exists)
   --GZIP             Use this if you want to create a sql.gz for export, instead of a sqlite DB
                      In addition, this collects additional debug info as a comment in the SQL.
   --OMNIVERSE        Selects ALL universes and exports(gzip) their JSON. (*) Do not specify --UNIVERSE.
   --DBFILE           [=] <output-file-name> (Optional. Generated if unspecified)

\x1b[1;33;100mADVANCED options\x1b[0m
   --HTTPCONNECT            [=] [curl | tiny]    (Optional. Whether to use 'curl' or HTTP::Tiny(Default))
   --FOLLOWER_LAG_MINIMUM   [=] <value> (milisec)(collect tablet follower lag for values >= this value(default 1000))
   --CONFIG_FILE_(PATH|NAME)[=] <path-or-name-of-file-containing-options> (i.e --CONFIG_FILE_PATH & .._NAME)

\x1b[1;33;100mBackfill related options\x1b[0m
   --WAIT_INDEX_BACKFILL        If specified, this program runs till backfills complete. No report or DB.
   --INDEX_NAME             [=] <idx-name> Optionally Used with WAIT_INDEX_BACKFILL, to specify WHICH idx to wait for.
   --SLEEP_INTERVAL_SEC     [=] nn  Number of seconds to sleep between check for backfill; default 30.

    If \x1b[1;30;43mSTDOUT\x1b[0m is redirected, it can be sent to  a SQL file, or gzipped, and collected for offline analysis.
    You may abbreviate option names up to the minimum required for uniqueness.
    Options can be set via --cmd-line, or via environment, or both, or via a "config_file".
    We look for config files by default at --CONFIG_FILE_PATH=/home/yugabyte with a name "*.yba.rc".
    Expected config file content format is : EXPORT <OPTION-NAME>="VALUE"
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
 package UniverseClass;
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
   CONFIG_FILE_PATH     => "/home/yugabyte/",
   CONFIG_FILE_NAME     => '.yba*.rc',
   CUSTOMER             => undef,
   WAIT_INDEX_BACKFILL  => 0,
   INDEX_NAME           => undef,
   SLEEP_INTERVAL_SEC   => 30,
   OMNIVERSE            => 0,  # If set, capture ALL universe's JSON
);

#---- Start ---
my ($YBA_API,$SQL_OUTPUT_FH, $db, $universe);
warn "-- ", $opt{STARTTIME_PRINTABLE}, " : Moses version $VERSION \@$opt{LOCALHOST} starting ...", "\n";
Initialize();

if ($opt{WAIT_INDEX_BACKFILL}){
   while ( Check_Index_Backfill_complete() ){
       sleep $opt{SLEEP_INTERVAL_SEC};
   }
   warn TimeDelta("Index backfill wait COMPLETED. Exiting.");
   exit 0;
}

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
$opt{DBFILE}=~/gz$/ and warn "\t To process into a DB, RUN: gunzip -c $opt{DBFILE} | sqlite3 "
                      . substr($opt{DBFILE},0,-7) . ".sqlite\n";
exit 0;
#----------------------------------------------------------------------------------------------
sub Get_and_Parse_tablets_from_tservers{

  my $prev_node_msg = "";

  for my $n (@{ $universe->{NODES} }){
      next unless $n->{isTserver};
      if ( $n->{state} ne  'Live'){
         warn "-- Node $n->{nodeName} $n->{Tserver_UUID} is $n->{state} .. skipping\n";
         next;
      }
      $n->{Tserver_UUID} ||= "*Unknown\@Idx-" . $n->{nodeIdx} . "*"; # Can happen for un-initialized system
      my $tabletCount = 0;
      print "SELECT '", TimeDelta("Processing tablets on $n->{nodeName} $n->{Tserver_UUID} ($n->{private_ip},Idx $n->{nodeIdx})... $prev_node_msg"),"';\n";
      my $html_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/tablets?raw","BASE_URL_UNIVERSE",1); # RAW
      if ( $opt{GZIP} ){
         # Save raw Node HTML  to the output as a comment, for debugging
         (my $escaped_HTML = $html_raw) =~s|\*/|*^/g|; # Escape closing comment 
         print "-- Raw Node  $n->{nodeName} $n->{Tserver_UUID} ($n->{private_ip},Idx $n->{nodeIdx}) HTML --\n/*\n",
                $escaped_HTML,"\n*/\n";
      }
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
          $t->{LEADER} and $leaders{$t->{LEADER}} ++;
          if ($t->{TABLET_UUID} eq "hash_split"){
             # Something wierd about this line - record it as a comment, and ignore line
             $db->putlog("WARNING: unexpected tablet line#$. for node $n->{private_ip}:$_",4);
             $db->Collect_Error_Metric("Unparsed Tablet line for node $n->{private_ip}");
             next;
          }
          $db->Insert_Tablet($t, $n->{Tserver_UUID});
      }
      close $f;
      Collect_Follower_Lag_metrics($n);
      Get_Node_Metrics($n);
      $db->putsql("END TRANSACTION; --Tablets for  tserver $n->{nodeName}");
      $db->putlog("Found $tabletCount tablets on $n->{nodeName}:"
          . join (",\n\t ",map{ " $leaders{$_} leaders  on $_" } sort keys %leaders));
      $prev_node_msg= "(Idx $n->{nodeIdx} had $tabletCount tablets, "
                    . ($leaders{$n->{private_ip}} || 0) . " leaders"
                    . ")";

  }
  print "SELECT '", TimeDelta("Completed Node Processing. $prev_node_msg"),"';\n";
  $db->putsql("CREATE UNIQUE INDEX tablet_idx ON tablet (node_uuid,tablet_uuid);");
}
#----------------------------------------------------------------------------------------------
sub Collect_Follower_Lag_metrics{
   my ($n) = @_;
   my $lags = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/metrics?metrics=follower_lag_ms","BASE_URL_UNIVERSE");
   my $ts = time();

  my $bj = JSON::Tiny::decode_json($YBA_API->{json_string});
   # Put stuff into metrics table 
  for my $metricInfo (@$bj){
     for my $m( @{$metricInfo->{metrics}} ){
      next unless $m->{value} >= $opt{FOLLOWER_LAG_MINIMUM};
       $db->putsql("INSERT INTO metrics VALUES("
             #timestamp INTEGER, node-uuid , tablet_id TEXT, metric_name TEXT, metric_value NUMERIC
                  . qq|'$m->{name}'|
                  . qq|,"$n->{Tserver_UUID}"|  # Node UUID 
                  . qq|,"tablet"|
                  . qq|,"$metricInfo->{id}"| # Tablet ID
                  . qq|,$m->{value}|
                  . ");");
     }
  }

}
#----------------------------------------------------------------------------------------------
sub Initialize{

    GetOptions (\%opt, qw[DEBUG! HELP! VERSION!
                        API_TOKEN|TOKEN=s YBA_HOST=s UNIVERSE=s
                        GZIP! DBFILE=s SQLITE=s GZIP! DROPTABLES!
                        HTTPCONNECT=s CURL=s FOLLOWER_LAG_MINIMUM=i
                        CONFIG_FILE_PATH=s CONFIG_FILE_NAME=s CUSTOMER=s
                        WAIT_INDEX_BACKFILL|WAITINDEXBACKFILL|WAITBACKFILL|WAIT_BACKFILL!
                        INDEX_NAME|INDEXNAME=s SLEEP_INTERVAL_SEC|INTERVAL=i
                        OMNIVERSE|COSMOS|KRAMER!]
               ) or die "ERROR: Invalid command line option(s). Try --help.";

    if ($opt{HELP}){
      warn $HELP_TEXT;
      exit 0;
    }
    $opt{VERSION} and exit 1; # Version request already fulfilled.
    
    #Process Config file(s) and extract APT_TOKEN, Univ etc..
    my @conf_files = glob "$opt{CONFIG_FILE_PATH}/$opt{CONFIG_FILE_NAME}"; # Try specified path...
    @conf_files or @conf_files = glob "./$opt{CONFIG_FILE_NAME}"; # Try current path
    for my $cfile_name (@conf_files){
        $opt{DEBUG} and print "--DEBUG: Reading $cfile_name\n";
        open my $f, "<", $cfile_name or die "ERROR: Opening $cfile_name:$!";
        while(<$f>){
           my ($key,$val) = m/\b(\w+)\s*=\s*["']?([\w\-:\.\/]+)['"]?/;
           next unless $key and  exists $opt{uc $key};
           $opt{uc $key} ||= $val; # Cmd-line overrides file info 
        }
        close $f;
    }

    for my $k (keys %opt){
       next if $opt{$k} ;     # Already has a value
       next unless $ENV{$k} ; #Env exists 
       $opt{$k} = $ENV{$k};   # Use Env value as last resort 
    }
    if ($opt{DEBUG}){
      print "-- ","DEBUG: Option $_\t="
          .(defined $opt{$_} ? ref $opt{$_} eq "ARRAY"? join (",",@{$opt{$_}}): $opt{$_}
                             : "*Not Defined*")
          .";\n" 
          for sort keys %opt;
    }
    # Initialize connection to YBA 
    $YBA_API = Web::Interface::->new();
   eval { $opt{YBA_JSON} = $YBA_API->Get("/customers","BASE_URL_API_V1") };
   if ($@  or  (ref $opt{YBA_JSON} eq "HASH" and $opt{YBA_JSON}{error})){
     die "ERROR:Unable to `get` YBA API customer info - Bad API_TOKEN?:$@"; 
   }
   ## All is well - we got the info in $opt{YBA_JSON}
   if (scalar(@{ $opt{YBA_JSON} }) == 1){
      $opt{CUST_UUID} = $opt{YBA_JSON}[0]{uuid}; # Simple - single cust.
   }elsif (not $opt{CUSTOMER}){
       warn "WARNING: --CUSTOMER is not specified, and multiple customers exist .. selecting First(".$opt{YBA_JSON}[0]{name}.").\n";
       $opt{CUST_UUID} = $opt{YBA_JSON}[0]{uuid};
   }else{
      for my $c(@{ $opt{YBA_JSON} }){
        $opt{DEBUG} and print "--DEBUG: CUSTOMER:$c->{uuid} = '$c->{name}'.\n";
         next unless $c->{uuid} eq $opt{CUSTOMER} or $c->{name} =~/$opt{CUSTOMER}/i;
         $opt{CUST_UUID} = $c->{uuid};
         last;
      }
      die "ERROR: Customer '$opt{CUSTOMER}' was not found (Run with --debug to list)" unless $opt{CUST_UUID}; 
   }
   $YBA_API->Set_Value("CUST_UUID", $opt{CUST_UUID});
   $opt{DEBUG} and print "--DEBUG: Customer $opt{CUST_UUID} selected.\n";
   
   $YBA_API->{"UNIV_UUID"} = undef; # We have not found it yet 
   $opt{UNIVERSE_LIST} = $YBA_API->Get("/customers/$YBA_API->{CUST_UUID}/universes","BASE_URL_API_V1");
   ref($opt{UNIVERSE_LIST}) eq "ARRAY" or die "ERROR: Could not get universe list. Bad API token ? --customer?";
   if ($opt{UNIVERSE}){
      my $u; # Try exact match
      ($u) = grep {$_->{universeUUID} eq $opt{UNIVERSE}} @{$opt{UNIVERSE_LIST}} and  $YBA_API->Set_Value("UNIV_UUID",$u->{universeUUID});
      ($u) = grep {$_->{name}         eq $opt{UNIVERSE}} @{$opt{UNIVERSE_LIST}} and  $YBA_API->Set_Value("UNIV_UUID",$u->{universeUUID});
   }
   for my $u (@{$opt{UNIVERSE_LIST}}){ # Try regex match
      $opt{DEBUG} and print "--DEBUG: Scanning Universe: $u->{name}\t $u->{universeUUID}\n";
      next if $opt{OMNIVERSE};
      last if $YBA_API->{"UNIV_UUID"}; # Already set 
      if ($opt{UNIVERSE}  and  $u->{name} =~/$opt{UNIVERSE}/i){
         $opt{DEBUG} and print  "-- Selected Universe $u->{name}\t $u->{universeUUID}\n";
         $YBA_API->Set_Value("UNIV_UUID",$u->{universeUUID});
         last;
      }
   }
   if ( $opt{OMNIVERSE} ){
      Capture_All_UNiverses_JSON();
      exit 0;
   }
   if (! $YBA_API->{"UNIV_UUID"}){
       warn "Please select a universe name (or unique part thereof) from:\n";
       warn "\t$_->{name}\n"  for (@{$opt{UNIVERSE_LIST}});
       die "ERROR: --UNIVERSE ($opt{UNIVERSE}) incorrect or unspecified\n";
   }
   # -- Universe  details -
   $universe = UniverseClass::->new($YBA_API) ; # $YBA_API->Get(""); # Huge Univ JSON 
  
  if ($universe->{name}){
     print TimeDelta(join("", "UNIVERSE: ", $universe->{name}," on ", $universe->{universeDetails}{clusters}[0]{userIntent}{providerType},
           " ver ",$universe->{universeDetails}{clusters}[0]{userIntent}{ybSoftwareVersion})),"\n";
  }else{
     die "ERROR: Universe info not found \n";
  }
  $universe->Check_Status(sub{warn "WARNING:$_[0]\n"});

  if (! $universe->{MASTER_LEADER_NODE}){
     die "ERROR: Cannot find Master/Leader for this universe";
  }

  if ($opt{WAIT_INDEX_BACKFILL}){
     return; # No need to create output etc...
  }

  #--- Initialize SQL output -----
  Setup_Output_Processing(); # Figure out if we are piping to sqlite etc.. 

  $db = DatabaseClass::->new(DROPTABLES=>$opt{DROPTABLES});
  
  # Put Univ and node info into DB
  $db->Insert_nodes($universe->{NODES});
  $db->CreateTable("gflags",qw|type key  value|);
  $opt{DEBUG} and print "SELECT '",TimeDelta("DEBUG:Extracting gflags..."),"';\n";
  $universe->Extract_gflags_into_DB($db);
  $universe->Check_placementModificationTaskUuid($db);
  
  # Get dump_entities JSON from MASTER_LEADER
  $opt{DEBUG} and print "SELECT '",TimeDelta("DEBUG:Getting Dump Entities..."),"';\n";  
  my $entities = # $YBA_API->Get("/proxy/$ml_node->{private_ip}:$master_http_port/dump-entities","BASE_URL_UNIVERSE");
                 $universe->Get_Master_leader_Endpoint_data("/dump-entities", 0);
  if ( $opt{GZIP} ){
    # Save raw univ & entities to the output as a comment, for debugging
    (my $escaped_JSON = $universe->{JSON_STRING}) =~s|\*/|*^/g|; # Escape closing comment 
    print "-- Universe JSON --\n/*\n", $escaped_JSON,"\n*/\n";
    ($escaped_JSON = $YBA_API->{json_string}) =~s|\*/|*^/g|; # Escape closing comment in ENTITIES
    print "-- ENTITIES --\n/*\n", $escaped_JSON,"\n*/\n";
  }
  # Analyze & save DUMP ENTITIES contained in  $YBA_API->{json_string} 
  Handle_ENTITIES_Data($entities);

  $db->CreateTable("metrics",qw|metric_name node_uuid entity_name entity_uuid|,"value NUMERIC");

  Handle_xCluster_Data(); # Uses Globals :$db,$universe;
  # Since we have SELECTed the sqlite file handle, we need funny-looking "print" statements
  # to get SQLITE to display our "progress" messages. (Old SQLITE does not support ".print", so we use SELECTs)
  $universe->Get_runtime_config_w_callback(undef, # Global scope 
              sub{ my ($cfg) = @_;
                   $db->putsql("INSERT INTO gflags VALUES ('RUNTIMECFG','$cfg->{key}','$cfg->{value}');");
              });
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
                                $universe->{name}, $opt{GZIP}?"sql.gz":'sqlite');
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
  # close STDOUT; # Don't close it - because it causes warnings later, as FH#1 can be reused
  select $SQL_OUTPUT_FH; # All subsequent "print" goes to this file handle.
}
#----------------------------------------------------------------------------------------------
sub Handle_ENTITIES_Data{
  my ($bj) = @_; # Entities decoded JSON 
  $opt{DEBUG} and print "--DEBUG:IN: Handle_ENTITIES_Data\n";
    $db->CreateTable("keyspaces","id TEXT PRIMARY KEY","name TEXT", "type TEXT"); # -- YCQL
    $db->CreateTable("tables","id TEXT PRIMARY KEY",qw| keyspace_id name state uuid tableTyp 
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
    for my $n (@{ $universe->{NODES} }){
       $n->{Tserver_UUID} = $node_by_ip{$n->{private_ip}}; # update in-mem info
    }
    $db->putsql( "UPDATE NODE "
               . "SET nodeUuid=(select server_uuid FROM ent_tablets "
              # . "WHERE  substr(addr,1,instr(addr,\":\")-1) = private_ip limit 1);\n");
               . "WHERE substr(addr,1,length(addr) - 5) = private_ip limit 1);\n");
    $db->putsql("END TRANSACTION; -- Entities");
}

#------------------------------------------------------------------------------------------------
sub Handle_xCluster_Data{
  # Uses Globals $db, $universe
  $db->CreateTable("xcluster",qw|uuid  name sourceUniverseUUID targetUniverseUUID status createTime modifyTime |);
  $db->CreateTable("xcTable" ,my @xcTableFields= qw|xcid table_uuid streamId replicationSetupDone needBootstrap indexTable status lag|);
  my %xcTableIdx = map { $xcTableFields[$_] => $_ } 0..$#xcTableFields; # array index map 
  my %xcConfig;

  $universe->Get_xCluster_details_w_callback(
    sub{
      my ($type, $xClusterDetails) = @_;
      my $uuid = $xClusterDetails->{uuid};
      $xcConfig{$uuid} and return; # Already seen this
      $xcConfig{$uuid} =  $xClusterDetails;
      $db->putsql("INSERT INTO xcluster VALUES('"
        . join("','", map {$xClusterDetails->{$_}} qw|uuid  name sourceUniverseUUID targetUniverseUUID status createTime modifyTime |)
        . "');"
      );
      for my $table_uuid(@{ $xClusterDetails->{tables} }){
         my @table_val;
         $table_val[$xcTableIdx{lag}] =  ref $xClusterDetails->{lag} ? 0 :  $xClusterDetails->{lag} + 0; 
         my ($tableDetail) = grep {$_->{tableId} eq $table_uuid} @{ $xClusterDetails->{tableDetails} };
         if ($tableDetail){
            for (keys %$tableDetail){
               next unless defined( my $idx = $xcTableIdx{$_} );
               $table_val[$idx] = $tableDetail->{$_};
            }
         }
         $table_val[$xcTableIdx{xcid}]   = $uuid;
         $table_val[$xcTableIdx{table_uuid}] = $table_uuid;
         $db->putsql("INSERT INTO xcTable VALUES('"
            . join("','", map {defined $table_val[$_]? $table_val[$_] : ''} 0..$#xcTableFields)
            . "');"
         );
      }
    }
  );
}

#------------------------------------------------------------------------------------------------
sub Get_Node_Metrics{
  my ($n) = @_; # NODE 
  my %post_process;
  
  my %metric_handler=(
     ql_read_latency    => sub{my ($m,$table,$val)=$_[1]=~/^(\w+)\{table_id="(\w+).+?\s(\d+)/ or return;
                               $post_process{$m}{$_[0]}{$table}=$val;},
     log_append_latency => sub{my ($m,$table,$val)=$_[1]=~/^(\w+)\{table_id="(\w+).+?\s(\d+)/ or return;
                               $post_process{$m}{$_[0]}{$table}=$val;}, #microseconds 
     ql_write_latency   => sub{my ($m,$table,$val)=$_[1]=~/^(\w+)\{table_id="(\w+).+?\s(\d+)/ or return;
                               $post_process{$m}{$_[0]}{$table}=$val;},                               
     server_uptime_ms   => sub{my ($m,$val)=$_[1]=~/^(\w+).+?\s(\d+)/;save_metric($m,$_[0],0,$val)},
     async_replication_ => sub{  # committed_lag_micros and sent_lag_micros
                              my ($m,$table_id,$val)=$_[1]=~/^(\w+).+table_id="(\w+)".+}\s*(\d+)/;
                              save_metric($m,$_[0],$table_id,$val);
                              },
     hybrid_clock_skew  => sub{my ($m,$val)=$_[1]=~/^(\w+).+?\s(\d+)/;save_metric($m,$_[0],0,$val)},
     'handler_latency_yb_tserver_TabletServerService_Read{quantile="p99' #microseconds
                        => sub{my ($val)=$_[1]=~/\s(\d+)/;save_metric('tserver_read_latency_p99',$_[0],0,$val)},
     'handler_latency_yb_tserver_TabletServerService_Write{quantile="p99'
                        => sub{my ($val)=$_[1]=~/\s(\d+)/;save_metric('tserver_write_latency_p99',$_[0],0,$val)},
  );
  my $regex = "(^" . join("|^",map {quotemeta} keys(%metric_handler)). ")";

  if ($n->{isTserver}){
      my $metrics_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/prometheus-metrics?reset_histograms=false",
                                      "BASE_URL_UNIVERSE",1); # RAW

      while($metrics_raw=~/$regex(.+)$/mg){
        $metric_handler{$1}-> ($n->{Tserver_UUID},"$1$2");
      }
  }
  if ($n->{isMaster}){
      my $metrics_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{masterHttpPort}/prometheus-metrics?reset_histograms=false",
                                      "BASE_URL_UNIVERSE",1); # RAW
      while($metrics_raw=~/$regex(.+$)/mg){
        $metric_handler{$1}-> ("Master-" . $n->{private_ip},"$1$2");
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

}
#------------------------------------------------------------------------------------------------
sub save_metric{
  my ($metric,$node_uuid,$table_uuid,$value)=@_;
  return unless defined $metric  and  defined $value;
  $db->putsql("INSERT INTO METRICS VALUES('$metric','$node_uuid','TABLE','$table_uuid',$value);");
}
#------------------------------------------------------------------------------------------------
sub Check_Index_Backfill_complete{
  $opt{DEBUG} and print TimeDelta("DEBUG:Getting Index backfill from $universe->{MASTER_LEADER_NODE}->{private_ip} ..."),"';\n";  
  my $task_list = $universe->Get_Master_leader_Endpoint_data("/tasks?raw",1);
  my $active_backfills = 0;
  Read_this_buffer_HTML_Table_w_callback(
    \$task_list,
    sub{ my ($v, $fref,$line,$row) = @_;
       return unless $v and $v->{JOB_NAME} and $v->{JOB_NAME} eq "Backfill Table";
       return if $v->{STATE} eq "kComplete";
       $active_backfills++;
       print "Backfill#$active_backfills: $v->{STATE} ",
              $v->{START_TIME}, ", running for ",
              $v->{DURATION},": $v->{DESCRIPTION}.\n";
       return unless $opt{INDEX_NAME};
       return if $v->{DESCRIPTION} =~m/\b$opt{INDEX_NAME}\b/i; # Target index is still being backfilled
       $active_backfills--; # THe index we were looking for is NOT in backfill list
    },
  );
  return $active_backfills;
}
#------------------------------------------------------------------------------------------------
sub Capture_All_UNiverses_JSON{ # Handles $opt{OMNIVERSE}
  warn TimeDelta("Getting JSON for " 
             . scalar(@{$opt{UNIVERSE_LIST}}) . " Universes..."), "\n";
  my $output_sqlite_dbfilename = $opt{DBFILE} ||=
      join(".", unixtime_to_printable($opt{STARTTIME},"YYYY-MM-DD"),$opt{LOCALHOST},"UniverseInfo","gz");

  open ($SQL_OUTPUT_FH, "|-", "gzip -c > $output_sqlite_dbfilename")
         or die "ERROR: Could not start gzip : $!";
  for my $u (@{$opt{UNIVERSE_LIST}}){
      $opt{DEBUG} and warn "--DEBUG: Scanning Universe: $u->{name}\t $u->{universeUUID}\n";
      $YBA_API->Set_Value("UNIV_UUID",$u->{universeUUID});
      $universe = UniverseClass::->new($YBA_API) ; # $YBA_API->Get(""); # Huge Univ JSON 
      
      $universe->{name} or  die "ERROR: Universe info not found for $u->{universeUUID} \n";
      print $SQL_OUTPUT_FH "-- Universe $u->{name} JSON --\n",  $universe->{JSON_STRING},"\n";

      if (my $dead_nodes = $universe->Check_Status(sub{})){
          warn "WARNING: $dead_nodes Nodes are not LIVE, in '$universe->{name}'. Skipping ENTITIES.\n";
      }else{
          my $entities = $universe->Get_Master_leader_Endpoint_data("/dump-entities", 0);
          print $SQL_OUTPUT_FH "-- ENTITIES --\n",  $YBA_API->{json_string},"\n\n";
      }
      print $SQL_OUTPUT_FH "\n"; # Extra blank line after universe data 
  }
  
  close $SQL_OUTPUT_FH;
  warn TimeDelta("COMPLETED. '$opt{DBFILE}' Created " , $opt{STARTTIME}),"\n";
}
#------------------------------------------------------------------------------------------------
sub Read_this_buffer_HTML_Table_w_callback{
   my ($buf_ref, $callback) = @_;
   open my $f,"<",$buf_ref or die "Cannot open buffer as file:$!";
   local $/= "</tr>\n"; # "Line" separator 
   my $row=0;

   my @fields; # = map{tr/ -/_/;uc } $header=~m{<th>([^<]+)</th>}sg;

   while(<$f>){
      if (m{</?table[^>]+>}){
          @fields=();
          $row = 0;
      }
      if (0 == scalar(@fields)  and  m{<tr><th>}m){
         @fields = map{tr/ -/_/;uc } m{<th>([^<]+)</th>}gm;
         next;
      }
      my $h=0;
      my %val = map{$fields[$h++] => defined $_?$_:''} $_=~m{<t[hd]>(.*?)</t[hd]>}gm; # Can have empty <td>'s
      $callback->(\%val,\@fields,$_,++$row);
   }
   close $f;
   $callback->(undef); # Ended
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

sub Append_Pending_Message{
  my ($self,$msg) = @_; # Save $msg for later output (with View summary)
  push @{ $self->{PENDING_MESSAGES} }, $msg;
}

sub Collect_Error_Metric{
  my ($self,$msg) = @_;
  $self->{ERROR_METRIC}{$msg}++;
}

sub Create_Views{
  my ($self) = @_;
  $self->putsql(<<"__SQL__");
  CREATE VIEW version_info AS 
    SELECT '$0' as program, '$VERSION' as version, '$opt{STARTTIME_PRINTABLE}' AS run_on, '$opt{LOCALHOST}' as host;

  CREATE VIEW tablets_per_node AS 
    SELECT node_uuid,min(public_ip) as node_ip,min(region ) as region,  count(*) as tablet_count,
           sum(CASE WHEN tablet.state='TABLET_DATA_COPYING' THEN 1 ELSE 0 END) as copying,
           sum(CASE WHEN private_ip = leader THEN 1 ELSE 0 END) as leaders,
           sum(CASE WHEN tablet.state = 'TABLET_DATA_TOMBSTONED' THEN 1 ELSE 0 END) as tombstoned,
           count(DISTINCT table_name) as table_count
  FROM tablet,node 
  WHERE isTserver  and node.nodeuuid=node_uuid 
  GROUP BY node_uuid
  UNION
    SELECT '~~TOTAL~~',
      '*(All '|| (select count(*) from node where isTserver=1) || ' nodes)*', 'ALL',
       (Select count(*) from tablet),(Select count(*) from tablet WHERE state='TABLET_DATA_COPYING'),
      (SELECT count(*) from tablet where leader > ""),
      (SELECT count(*) from tablet where state = 'TABLET_DATA_TOMBSTONED'),
      (SELECT count(DISTINCT table_name) as table_count from tablet)
  ORDER BY 1;

  CREATE VIEW tablet_replica_detail AS
    SELECT t.namespace,t.table_name,t.table_uuid,t.tablet_uuid,
  count(DISTINCT LEADER) as leader_count, count(*) as replicas
  from tablet t
  WHERE t.state != 'TABLET_DATA_TOMBSTONED'
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
        || ' -certs_dir_name \$TLSDIR;sleep 10;# '
        || trd.replicas || ' replica(s)'
        AS cmd_to_run
   from tablet t,node ,tablet_replica_detail trd
   WHERE  node.isTserver  AND nodeuuid=node_uuid
         AND  t.tablet_uuid=trd.tablet_uuid  
       AND trd.leader_count !=1
       AND t.state = 'RUNNING';
  CREATE VIEW IF NOT EXISTS table_sizes AS 
  SELECT T.NAMESPACE,T.TABLE_NAME,count(*) as tablets,RF1_tablets,
     sum(T.NUM_SST_FILES) as sst_files, -- D.sst_files as RF1_SST_files,
     round(sum(T.SST_FILES) /1024.0/1024.0/1024.0,2) as sst_gb,
     round(sum(T.SST_FILES_UNCOMPRESSED) /1024.0/1024.0/1024.0,2) as sst_gb_uncompr,
     round(sum(T.WAL_FILES) /1024.0/1024.0/1024.0,2) as wal_GB,
     round(sum(T.TOTAL) /1024.0/1024.0/1024.0,2) as tot_gb,
     D.tot_gb as RF1_tot_GB
  FROM TABLET T,
      (SELECT NAMESPACE,TABLE_NAME,count(*) as RF1_tablets,
       CAST(round(sum(NUM_SST_FILES),0) AS INTEGER) as sst_files,
       round(sum(SST_FILES) /1024.0/1024.0/1024.0,2) as sst_gb,
       round(sum(SST_FILES_UNCOMPRESSED) /1024.0/1024.0/1024.0,2) as sst_gb_uncompr,
       round(sum(WAL_FILES) /1024.0/1024.0/1024.0,2) as wal_GB,
       round(sum(TOTAL) /1024.0/1024.0/1024.0,2) as tot_gb
    FROM (SELECT  tablet_uuid,
            NAMESPACE,TABLE_NAME,TABLE_UUID, avg(NUM_SST_FILES) NUM_SST_FILES ,avg(SST_FILES) SST_FILES ,
           avg(SST_FILES_UNCOMPRESSED) SST_FILES_UNCOMPRESSED,avg(TOTAL) TOTAL ,avg(WAL_FILES ) WAL_FILES
         FROM tablet GROUP BY tablet_uuid, NAMESPACE,TABLE_NAME,TABLE_UUID) 
    GROUP BY NAMESPACE,TABLE_NAME) D 
  WHERE t.NAMESPACE=d.NAMESPACE AND T.TABLE_NAME=D.TABLE_NAME
  GROUP BY T.NAMESPACE,T.TABLE_NAME;
  
  CREATE VIEW IF NOT EXISTS namespace_sizes AS 
  SELECT T.NAMESPACE,count(*) as tablets,RF1_tablets,
     sum(T.NUM_SST_FILES) as sst_files, -- D.sst_files as RF1_SST_files,
     round(sum(T.SST_FILES) /1024.0/1024.0/1024.0,2) as sst_gb,
     round(sum(T.SST_FILES_UNCOMPRESSED) /1024.0/1024.0/1024.0,2) as sst_gb_uncompr,
     round(sum(T.WAL_FILES) /1024.0/1024.0/1024.0,2) as wal_GB,
     round(sum(T.TOTAL) /1024.0/1024.0/1024.0,2) as tot_gb,
     D.tot_gb as RF1_tot_GB
  FROM TABLET T,
      (SELECT NAMESPACE,TABLE_NAME,count(*) as RF1_tablets,
       CAST(round(sum(NUM_SST_FILES),0) as INTEGER) as sst_files,
       round(sum(SST_FILES) /1024.0/1024.0/1024.0,2) as sst_gb,
       round(sum(SST_FILES_UNCOMPRESSED) /1024.0/1024.0/1024.0,2) as sst_gb_uncompr,
       round(sum(WAL_FILES) /1024.0/1024.0/1024.0,2) as wal_GB,
       round(sum(TOTAL) /1024.0/1024.0/1024.0,2) as tot_gb
    FROM (SELECT  tablet_uuid,
            NAMESPACE,TABLE_NAME,TABLE_UUID, avg(NUM_SST_FILES) NUM_SST_FILES ,avg(SST_FILES) SST_FILES ,
           avg(SST_FILES_UNCOMPRESSED) SST_FILES_UNCOMPRESSED,avg(TOTAL) TOTAL ,avg(WAL_FILES ) WAL_FILES
         FROM tablet GROUP BY tablet_uuid, NAMESPACE,TABLE_NAME,TABLE_UUID) 
    GROUP BY NAMESPACE) D 
  WHERE t.NAMESPACE=d.NAMESPACE 
  GROUP BY T.NAMESPACE;
-- tablet_view with isLeader
CREATE VIEW tablet_view AS
SELECT
    t.node_uuid,
    t.TABLET_UUID,
    t.NAMESPACE,
    t.TABLE_NAME,
    t.TABLE_UUID,
    t.STATE,
    t.HIDDEN,
    t.LEADER,
    t.FOLLOWERS,
    t.NUM_SST_FILES,
    t.PARTITION,
    t.LAST_STATUS,
    t.SST_FILES,
    t.SST_FILES_UNCOMPRESSED,
    t.TOTAL,
    t.WAL_FILES,
    CASE 
        WHEN t.LEADER = n.private_ip THEN 1
        ELSE 0
    END AS is_leader
FROM
    tablet t,
    node n
WHERE
    t.node_uuid = n.nodeUuid AND
    t.state = 'RUNNING';
-- Tablet count per size range per table
CREATE VIEW tablet_count_per_size_range AS
SELECT
    TABLE_NAME,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 < 2048 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 <2048 THEN 1 ELSE 0 END) END AS LT2GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 2048 AND 3072 THEN 1 ELSE 0 END) END AS s2GB_3GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 3072 AND 4096 THEN 1 ELSE 0 END) END AS s3GB_4GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 4096 AND 6144 THEN 1 ELSE 0 END) END AS s4GB_6GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 6144 AND 8192 THEN 1 ELSE 0 END) END AS s6GB_8GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 8192 AND 10240 THEN 1 ELSE 0 END) END AS s8GB_10GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 10240 AND 12288 THEN 1 ELSE 0 END) END AS s10GB_12GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 12288 AND 14336 THEN 1 ELSE 0 END) END AS s12GB_14GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 14336 AND 16384 THEN 1 ELSE 0 END) END AS s14GB_16GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 16384 AND 20480 THEN 1 ELSE 0 END) END AS s16GB_20GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 20480 AND 24576 THEN 1 ELSE 0 END) END AS s20GB_24GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 24576 AND 28672 THEN 1 ELSE 0 END) END AS s24GB_28GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 28672 AND 32768 THEN 1 ELSE 0 END) END AS s28GB_32GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 32768 AND 36864 THEN 1 ELSE 0 END) END AS s32GB_36GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 BETWEEN 36864 AND 40960 THEN 1 ELSE 0 END) END AS s36GB_40GB,
    CASE WHEN SUM(CASE WHEN TOTAL/1024/1024 > 40960 THEN 1 ELSE 0 END) = 0 THEN NULL ELSE SUM(CASE WHEN TOTAL/1024/1024 > 40960 THEN 1 ELSE 0 END) END AS GT40GB
FROM
    tablet,node
WHERE
    nodeUuid = node_uuid AND private_ip = leader
GROUP BY
    TABLE_NAME
ORDER BY
    2 DESC;
-- "list" mode ("|" delimited) for these single-line , ASCII highlighted output renders properly (Column mode does not)
.mode list
 SELECT char(27) ||  '[1;40;36;4m-- The following reports are available --' || char(27) || '[0;2m' ; -- Cyan on Black,UL - then FAINT
.tables
SELECT  char(27) || '[40;1;93;4m -- S u m m a r y --' || char(27) || '[0m' ; -- Bright yellow on black,UL

 CREATE VIEW IF NOT EXISTS Summary AS 
 SELECT   (SELECT count(*) from node) || ' Nodes ('
       || (SELECT count(DISTINCT region) from node) || ' Regions, '
       || (SELECT count(DISTINCT az) from node) || ' AZs), '
       || (SELECT count(*) from tablet)||' Tablets ('
       || (SELECT count(*) from tablet WHERE state='TABLET_DATA_TOMBSTONED') || ' Tombstoned, '
       || (SELECT count(*)  from leaderless) || ' Leaderless). '
       || (select count(*) from metrics) || ' metrics.'  AS Summary;
 
 SELECT * from Summary;

 SELECT tablet_count ||' tablets have '||replicas || ' replicas.' FROM tablet_replica_summary;
 SELECT char(27) || '[93;1mWARNING:' || char(27) || '[0m ' || nodeName||'('|| node_uuid || ') has '|| metric_name || '='||value  
         || ' (microseconds)'
 FROM  metrics,node
 WHERE node_uuid=nodeUuid and  node_uuid like 'Master-%' and metric_name='log_append_latency_avg' 
       and value+0 >= $opt{FOLLOWER_LAG_MINIMUM} * 1000;
 SELECT char(27) || '[93;1mWARNING:' || char(27) || '[0m' || ' Node ' || nodeName||'('|| node_uuid || ') has '|| metric_name || '='||value  
         || ' (microseconds)'
 FROM  metrics,node
 WHERE node_uuid=nodeUuid and metric_name IN ('hybrid_clock_skew')
      and value+0 >= $opt{FOLLOWER_LAG_MINIMUM}; -- Treat this as microsec for this metric
 SELECT  char(27) || '[93;1mWARNING:' || char(27) || '[0m' ||' Node ' || nodeName||'('|| node_uuid || ') has '|| metric_name || '='||value 
        || CASE WHEN entity_uuid != '0' THEN ' (' || entity_name || ' ' || entity_uuid || ')'
           ELSE '' END || ' (microseconds)'
 FROM  metrics,node 
 WHERE node_uuid=nodeUuid and metric_name IN ('tserver_read_latency_p99', 'tserver_write_latency_p99',
                       'async_replication_committed_lag_micros','async_replication_sent_lag_micros')
      and value+0 >=  $opt{FOLLOWER_LAG_MINIMUM}*1000;

.mode column
__SQL__

  for my $msg(@{ $self->{PENDING_MESSAGES} }){
      $self->putsql("SELECT '$msg';");
  }
  for my $k(sort keys %{ $self->{ERROR_METRIC} }){
      $self->putsql("SELECT ". $self->{ERROR_METRIC}{$k}  .",'$k';");
  }  
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
    BYTES   => 1,
    B       => 1,
    KB      => 1024,
    K       => 1024,
    MB      => 1024*1024,
    M       => 1024*1024,
    GB      => 1024*1024*1024,
    G       => 1024*1024*1024,
    TB      => 1024*1024*1024*1024,
    T       => 1024*1024*1024*1024,
  );

sub new_from_tr{
    my ($class, $line) = @_;
    # This parses a <tr> block for one tablet, extracts fields and returns a tablet object
    my $h=0;
    my %val = map{$fields[$h++] => defined $_?$_:''} $line=~m{<td>(.*?)</td>}gs; # Can have empty <td>'s
    
    ($val{TABLET_UUID}) = $val{TABLET_ID} =~m/^(?:<.+?>)?(\w+)/; # Usually <a href>, but tombstones have only uuid
    $val{FOLLOWERS} = join ",", $val{'RAFTCONFIG'} =~m/FOLLOWER: ([^<]+)/g ; # a CSV string 
    ($val{LEADER})    =  $val{'RAFTCONFIG'} =~m/LEADER: ([^<]+)/;
    my %disk = $val{ON_DISK_SIZE} =~m/>(\w.+?): ([^<]+)/g;

    $val{do{tr/ -/__/;uc }} = $disk{$_} for keys %disk;
        # Convert disk values to numeric (xlate K, B etc)
    for my $k (qw|SST_FILES SST_FILES_UNCOMPRESSED TOTAL WAL_FILES|){
        if ($val{$k}){
          my ($numeric,$unit) = $val{$k} =~m/([\-\.\d]+)(\w+)/;
          $val{$k} = $numeric *  ($kilo_multiplier{ uc $unit } || 1);
        }else{
          $val{$k} = 0;
        }
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
   return join (",", map {defined($self->{$_}) or $self->{$_}=""; $is_numeric{$_} ? $self->{$_}||0 : "'$self->{$_}'"} @db_fields);
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
    $opt{YBA_HOST} =~s|/$||; # Zap trailing slash 
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
                         ,verify_SSL         => 0, 
                      });
    if ($HTTP::Tiny::VERSION < 0.05){
       print "--WARNING: HTTP::Tiny version ", $HTTP::Tiny::VERSION, " is too old. Using CURL\n";
       $opt{HTTPCONNECT} = "curl";
       return $class->new(); # recurse
    }
    return $self;
}

sub Get{
    my ($self, $endpoint, $base, $raw) = @_;
    $self->{json_string}= "";
    my $url = $base ? $self->{$base} : $self->{BASE_URL_API_CUST_UNIV};
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
    $self->{BASE_URL_API_CUSTOMER} = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/api/v1/customers/$self->{CUST_UUID}";
    $self->{BASE_URL_API_CUST_UNIV}= "$self->{HTTP_PREFIX}$self->{YBA_HOST}/api/customers/$self->{CUST_UUID}/universes/$self->{UNIV_UUID}";
    $self->{BASE_URL_UNIVERSE}     = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/universes/$self->{UNIV_UUID}";
    $self->{BASE_URL_API_V1}       = "$self->{HTTP_PREFIX}$self->{YBA_HOST}/api/v1";
}

} # End of  package Web::Interface;  
#==============================================================================
BEGIN{
package UniverseClass;

use strict;
use warnings;

sub new{
  my ($class, $yba_api) = @_;
  $yba_api or die "YBA API Parameter is required";
  my $self = $yba_api->Get(""); # Perl-ized Huge Univ JSON
  $self->{JSON_STRING} = $yba_api->{json_string}; # Raw JSON string 
  $self->{YBA_API} = $yba_api;
  $opt{DEBUG} and print "--DEBUG:UNIV: $_\t","=>",$self->{$_},"\n" for qw|name creationDate universeUUID version |;
  _Extract_nodes($self);
  _Get_Master_Leader($self);
  for my $region (@{ $self->{universeDetails} {clusters} [0]{placementInfo}{cloudList}[0]{regionList} }){
      my $preferred = 0;
      my $az_node_count = 0;
      for my $az ( @{ $region->{azList} } ){
          $az->{isAffinitized} and $preferred++;
          $az_node_count += $az->{numNodesInAZ};
      }
      $self->{REGION}{$region->{name}}{PREFERRED}     = $preferred;
      $self->{REGION}{$region->{name}}{UUID}          = $region->{uuid};
      $self->{REGION}{$region->{name}}{AZ_NODE_COUNT} = $az_node_count;
      $opt{DEBUG} and print "--DEBUG:REGION $region->{name}: PREFERRED=$preferred, $az_node_count nodes, $region->{uuid}.\n";
  }
  return bless $self, $class;
}

#----------------------------------------------------------------------------------------------

sub Check_Status{
  my ($self, $error_callback) = @_;
  $self->{UNIV_STATUS} = $self->{YBA_API}->Get("/status")
    or  die "ERROR: Cannot get Universe status" ; # Fatal error - do not callback
  my $bad_nodes = 0;
  for my $node_name(keys %{ $self->{UNIV_STATUS} }){
      next if $node_name eq "universe_uuid"; # We already know this
      next if (my $node_status = $self->{UNIV_STATUS}{$node_name}{node_status}) eq "Live";
      $error_callback->("Node $node_name node_status=$node_status");
      $bad_nodes++;
  }
  return $bad_nodes;
}

sub Get_xCluster_details_w_callback{
  my ($self,$callback) = @_;
  for my $xcType (qw|targetXClusterConfigs sourceXClusterConfigs|){
     for my $xcUUID (@{ $self->{universeDetails}{$xcType} }){
         $opt{DEBUG} and print "--DEBUG:xCluster $xcType id=$xcUUID \n";
         # Get xCluster detail config for this ID
         my $xClusterDetails = $self->{YBA_API}->Get("/xcluster_configs/$xcUUID","BASE_URL_API_CUSTOMER");
         $callback->($xcType, $xClusterDetails);
     }
  }
}


sub _Extract_nodes{
    my ($self) = @_;
  
  $self->{NODES} = [];
  my $count=0;
  for my $n (@{  $self->{universeDetails}{nodeDetailsSet} }){
       push @{ $self->{NODES} }, my $thisnode = {map({$_=>$n->{$_}||''} qw|nodeIdx nodeName nodeUuid azUuid isMaster
                                       isTserver ysqlServerHttpPort yqlServerHttpPort state tserverHttpPort 
                     tserverRpcPort masterHttpPort masterRpcPort nodeExporterPort|),
                                map({$_=>$n->{cloudInfo}{$_}} qw|private_ip public_ip az region |) };
       $thisnode->{$_} =~tr/-//d for grep {/uuid/i} keys %$thisnode;
       $count++;
    }
    return $self->{NODES};  
}

sub _Get_Master_Leader{
  my ($self) = @_;
  # Find Master/Leader 
  $self->{MASTER_LEADER_NODE} = undef;
  my $leader_IP  = $self->{YBA_API}->Get("/leader")->{privateIP};
  if (! $leader_IP ){
    warn "WARNING:Could not get Master/Leader:\n\t" . $YBA_API->{json_string};
    return undef;
  }
  $opt{DEBUG} and print "--DEBUG:Master/Leader JSON:",$YBA_API->{json_string},". IP is $leader_IP .\n";
  ( $self->{MASTER_LEADER_NODE} ) = grep {$_->{private_ip} eq $leader_IP } @{ $self->{NODES} } or die "ERROR : No Master/Leader NODE found for $leader_IP ";
  my $master_http_port = $self->{universeDetails}{communicationPorts}{masterHttpPort} or die "ERROR: Master HTTP port not found in univ JSON";
}

sub Get_Master_leader_Endpoint_data{
  my ($self, $endpoint, $RAW) = @_;
  #("/proxy/$ml_node->{private_ip}:$master_http_port/dump-entities","BASE_URL_UNIVERSE");
  my $master_http_port = $self->{universeDetails}{communicationPorts}{masterHttpPort};
  return $self->{YBA_API}->Get("/proxy/$self->{MASTER_LEADER_NODE}->{private_ip}:$master_http_port$endpoint","BASE_URL_UNIVERSE",$RAW); # Get RAW data
}


#------------------------------------------------------------------------------------------------
sub Extract_gflags_into_DB{
  my ($self, $db) = @_;

  for my $k (qw| platformVersion universeUUID ybcSoftwareVersion|){
     next unless defined ( my $v= $self->{$k} );
     $db->putsql("INSERT INTO gflags VALUES ('CLUSTER','$k','$v');");
  }
  for my $k (qw| uuid clusterType |){
     next unless defined ( my $v= $self->{universeDetails}{clusters}[0]{$k} );
     $db->putsql("INSERT INTO gflags VALUES ('CLUSTER','$k','$v');");
  }
  for my $k (qw|universeName provider providerType replicationFactor numNodes ybSoftwareVersion enableYCQL
               enableYSQL enableYEDIS nodePrefix instanceType useSystemd useTimeSync|){
     next unless defined ( my $v= $self->{universeDetails}{clusters}[0]{userIntent}{$k} );
     $db->putsql("INSERT INTO gflags VALUES ('CLUSTER','$k','$v');");
  }
  for my $flagtype (qw|masterGFlags tserverGFlags |){
     next unless my $flag = $self->{universeDetails}{clusters}[0]{userIntent}{$flagtype};
     for my $k(sort keys %$flag){
        (my $v = $flag->{$k}) =~tr/'/~/; # Zap potential single quote in gflag value 
        $db->putsql("INSERT INTO gflags VALUES ('$flagtype','$k','$v');");
     }
  }
  for my $flagtype (qw|MASTER TSERVER |){ # New gflag location for 2.18
     next unless my $flag = $self->{universeDetails}{clusters}[0]{userIntent}{specificGFlags};
     next unless $flag = $flag->{perProcessFlags}{value}{$flagtype};
     for my $k(sort keys %$flag){
        (my $v = $flag->{$k}) =~tr/'/~/; # Zap potential single quote in gflag value 
        $db->putsql("INSERT INTO gflags VALUES ('$flagtype','$k','$v');");
     }
  }
}

sub Check_placementModificationTaskUuid{
  my ($self,$db) = @_;

  return unless  $self->{placementModificationTaskUuid};
  $db->putlog("Found placementModificationTaskUuid in Universe: ". $self->{placementModificationTaskUuid});
  $db->Append_Pending_Message("WARNING: Universe $self->{name} has a pending  placementModification Task - Please get assiance to clear it.");
}

sub Get_runtime_config_w_callback{
  my ($self, $scope,$callback) = @_;

  $scope ||= '00000000-0000-0000-0000-000000000000'; # Global
  my $config= $self->{YBA_API}->Get("/runtime_config/$scope","BASE_URL_API_CUSTOMER");
  #http://35.247.114.189/api/v1/customers/8ebe5e15-1ce5-425f-9f7d-1b3a68186554/runtime_config/00000000-0000-0000-0000-000000000000
  #{"type":"GLOBAL","uuid":"00000000-0000-0000-0000-000000000000","mutableScope":false,"configEntries":[]}
  $callback->($_) for @{ $config->{configEntries} };
}

1;
} # End of  package UniverseClass
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