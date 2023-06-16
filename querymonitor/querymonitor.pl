#!/usr/bin/perl

our $VERSION = "1.13";
my $HELP_TEXT = << "__HELPTEXT__";
#    querymonitor.pl  Version $VERSION
#    ===============
# Monitor running queries
# collect gzipped JSON file for offline analysis

__HELPTEXT__
use strict;
use warnings;
use Getopt::Long;
use Fcntl qw(:DEFAULT :flock);
use POSIX qw/setsid/;
use HTTP::Tiny;
{# Forward local-package declarations
  package Analysis::Mode;
  package Web::Interface; # Handles communication with YBA API 
  package MIME::Write::Simple;
  package MIME::Multipart::ParseSimple;
  package JSON::Tiny;
}

my %option_specs=( # Specifies info for globals saved in %opt. TYPE=>undef means these are INTERNAL, not settable.
    API_TOKEN     =>{TYPE=>'=s', DEFAULT=> $ENV{API_TOKEN}, HELP=>"From User profile."},
    YBA_HOST      =>{TYPE=>'=s', DEFAULT=> $ENV{YBA_HOST},  HELP=>"From User profile."},
    CUST_UUID     =>{TYPE=>'=s', DEFAULT=> $ENV{CUST_UUID}, HELP=>"From User profile."},  
    UNIV_UUID     =>{TYPE=>'=s', DEFAULT=> $ENV{UNIV_UUID}, HELP=>"From User profile."},
	HOSTNAME      =>{TYPE=>undef,DEFAULT=>do{chomp(local $_=qx|hostname|); $_} },
    STARTTIME     =>{TYPE=>undef,DEFAULT=> unixtime_to_printable(time(),"YYYY-MM-DD HH:MM")},
    STARTTIME_TZ  =>{TYPE=>undef,DEFAULT=> unixtime_to_printable(time(),"YYYY-MM-DD HH:MM","Include tz offset")},
    INTERVAL_SEC  =>{TYPE=>'=i', DEFAULT=> 5, HELP=>"Interval (Seconds) between  query snapshots"},
    RUN_FOR       =>{TYPE=>'=s', DEFAULT=> "4h", ALT=>"RUNFOR", HELP=>"Terminate after this much time. Must be in s(ec), m(in), h(our) etc."},
    ENDTIME_EPOCH =>{TYPE=>undef,DEFAULT=> 0,}, # Calculated after options are processed
    CURL          =>{TYPE=>'=s', DEFAULT=> "curl", HELP=>"Full path to curl command"},
    FLAGFILE      =>{TYPE=>'=s', DEFAULT=> "querymonitor.defaultflags", HELP=>"Name of file containing this program's options (--xx)"},
    OUTPUT        =>{TYPE=>'=s', DEFAULT=> undef, HELP=>"Output File name. Defaults to queries.<YMD>.<Univ>.mime.gz. Can specify STDOUT in ANALYZE mode"},
    DEBUG         =>{TYPE=>'!',  DEFAULT=> 0,},
    HELP          =>{TYPE=>'!',  DEFAULT=> 0,},
    VERSION       =>{TYPE=>'!',  DEFAULT=> 0,},
    DAEMON        =>{TYPE=>'!',  DEFAULT=> 1,HELP=>"Only for debugging(--NODAEMON)"},
    LOCKFILE      =>{TYPE=>'=s', DEFAULT=> "/var/lock/querymonitor.lock", HELP=>"Name of lock file"}, # UNIV_UUID will be appended
    LOCK_FH       =>{TYPE=>undef,DEFAULT=> undef,},
    MAX_QUERY_LEN =>{TYPE=>'=i', DEFAULT=> 2048, ALT=>"MAXQL|MAXL",},
    MAX_ERRORS    =>{TYPE=>'=i', DEFAULT=> 10,},
    SANITIZE      =>{TYPE=>'!',  DEFAULT=> 0,   HELP=>"Remove PII by truncating to WHERE clause"},
    ANALYZE       =>{TYPE=>'=s', DEFAULT=> undef, ALT=>"PROCESS", HELP=>"ANALYSIS mode Input File-name ('..csv.gz' ) to process through sqlite"},
    DB            =>{TYPE=>'=s', DEFAULT=> undef,     HELP=>"Analysis mode output SQLITE database file name. Automatic, but you can overrided here"},
    SQLITE        =>{TYPE=>'=s', DEFAULT=> "sqlite3", HELP=>"path to Sqlite binary"},
    UNIVERSE      =>{TYPE=>undef,DEFAULT=> undef,  },   # Universe detail info (Populated in initialize)
    HTTPCONNECT   =>{TYPE=>'=s', DEFAULT=> "curl",   HELP=>"How to connect to the YBA : 'curl', or 'tiny' (HTTP::Tiny)"},
    USETESTDATA   =>{TYPE=>'!',  DEFAULT=> undef,   HELP=>"TESTING only !! - Do not use."},
    TZOFFSET      =>{TYPE=>undef,DEFAULT=> undef,  },# This is set inside 'unixtime_to_printable', on first use 
    RPCZ          =>{TYPE=>'!',  DEFAULT=> 1,     HELP=>"If set, get query from each node, instead of /live_queries"},
);
my %opt = map {$_=> $option_specs{$_}{DEFAULT}} keys %option_specs;

my $quit_daemon = 0;
my $loop_count  = 0;
my $error_counter=0;
my $YBA_API;  # Populated in `Initialize` to a Web::Interface object
my $output;   # Populated in `Initialize` to a MIME::Write::Simple object

Initialize();

if ($opt{ANALYZE}){
   my $anl = Analysis::Mode::->new();
   $anl->Process_file_and_create_Sqlite();
   exit 0;
}

daemonize();

#------------- M a i n    L o o p --------------------------
while (not ($quit_daemon  or my $this_iter_ts=time() > $opt{ENDTIME_EPOCH} )){  # Infinite loop ...
   $loop_count++;
   my $next_iter_ts = $this_iter_ts + $opt{INTERVAL_SEC};
   Main_loop_Iteration();
   my $sleep_sec = $next_iter_ts - time();
   $sleep_sec > 0 and  sleep($sleep_sec);
}
#------------- E n d  M a i n    L o o p ---------------
# Could get here if a SIGNAL is received
warn(unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS") ." Program $$ Completed after $loop_count iterations.\n"); 
$output->Write_event(time(),unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS") ." Program $$ Completed after $loop_count iterations.",1); 
$opt{LOCK_FH} and close $opt{LOCK_FH} ;  # Should already be closed and removed by sig handler
unlink $opt{LOCKFILE};
$output->Close("FINAL");

exit 0;
#------------------------------------------------------------------------------
#==============================================================================
#------------------------------------------------------------------------------
sub Main_loop_Iteration{

    my $query_type = "Unknown";

    if ($opt{DEBUG}){
        print "--DEBUG: ",unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS")," Start main loop iteration $loop_count\n";
    }

    if ($opt{RPCZ}){
		Get_RPCZ_from_nodes(); # Write output directly 
		return 0;
	}
	my $queries = $YBA_API->Get("/live_queries");
	
    my $ts = time();
	if ($queries->{error}){
		$error_counter++ >= $opt{MAX_ERRORS} and $quit_daemon = 1;
	    return $queries->{error};	
	}
    for my $type (qw|ysql ycql|){
       for my $q (@{ $queries->{$type}{queries} }){
		   for my $subquery (split /;/, $q->{query}){
			 #Sanitize PII from each
			 my $sanitized_query = $opt{SANITIZE} ? SQL_Sanitize($subquery) : $subquery;
			 $sanitized_query=~s/^\s+//; # Zap leading spaces 
             #print join(",",$ts, $type,$q->{nodeName},$q->{id},$subquery),"\n";
			 $output->WriteQuery($ts, $type, $q, $sanitized_query);
		   }
       }
    }

  return 0;
}
#------------------------------------------------------------------------------
sub PrintDeep{
  my ($v, $txt, $level) = @_;
  $level ||= 0;
  $level >0 and print ", ";
  $txt and print "$txt";
        if (not ref $v){ #SCALAR
           print "$v";
        }elsif (ref $v eq "HASH"){
           print " \{";
           $level=0;
           PrintDeep($v->{$_},"$_=>",$level++)for sort keys %$v;
           print "\}";
        }elsif (ref $v eq "ARRAY"){
           print " \[";
           $level=0;
           PrintDeep($_,"",$level++) for  @$v;
           print "\] ";
        }else{
           die "ERROR: Invalid type:",ref($v),"\n";
        }
}
#-----
sub Get_RPCZ_from_nodes{

   for my $n (@{ $opt{NODES} }){
       if ($opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{enableYCQL}){
		    my $q =  $YBA_API->Get("/proxy/$n->{private_ip}:$n->{yqlServerHttpPort}/rpcz","BASE_URL_UNIVERSE");
			my $ts = time();
			$opt{DEBUG} and keys %$q and $loop_count<50 and  PrintDeep($q,"DEBUG:got ycql\@$ts:",0), print "\n";
			next unless my $conn_array=$q->{inbound_connections};

			for my $c (@$conn_array){
			   	next unless my $item_array = $c->{calls_in_flight};
				for my $item(@$item_array){
					my $info= {query => join(";",map{$_->{sql_string}}
					                              @{$item->{cql_details}{call_details}}),
					           type=>$item->{cql_details}{type},
					           elapsed_millis => $item->{elapsed_millis},
							   nodeName=>$n->{nodeName}};
					next unless length($info->{query}) > 1;
					$info->{remote_ip} = $c->{remote_ip};
					$output->WriteQuery($ts, "ycql", $info, $info->{query});
				}
			}
	   }

	   if ($opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{enableYSQL}){
		    my $q = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{ysqlServerHttpPort}/rpcz","BASE_URL_UNIVERSE");
            my $ts = time();
	        for my $c (@{$q->{connections}}){
		        next unless $c->{query};
				$c->{host} = $n->{nodeName};
				$opt{DEBUG} and $loop_count<50 and PrintDeep($c,"DEBUG:got ysql \@$ts:"), print "\n";
				$output->WriteQuery($ts, "ysql", $c, $c->{query});
	        }
	   }
   }

   return ;   
}	
#------------------------------------------------------------------------------
sub SQL_Sanitize{ # Remove PII 
   my ($q) = @_;
    # Remove leading spaces
   $q=~s/^\s+//;
    # For INSERT, remove VALUES
	# For SELECT, remove WHERE clause
   $q=~s/ values .+//gi;
   $q=~s/( where\s+\S[^=<>!"'\s]+).*/$1/i; # Include Var name after WHERE, but not value 
   return $q;
}
#------------------------------------------------------------------------------


#------------------------------------------------------------------------------
sub Initialize{
  print "-- ",$opt{STARTTIME_TZ}," Starting $0 version $VERSION  PID $$ on $opt{HOSTNAME}\n";
  my %flags_used;
  Getopt::Long::GetOptions (\%flags_used, 
                            map {my $t;my $alt=$option_specs{$_}{ALT};$alt=$alt?"|$alt":"";
                                 defined($t=$option_specs{$_}{TYPE}) ? ("$_$alt$t"):()} 
							keys %option_specs )
      or die "ERROR: Bad Option\n";
  $opt{$_} = $flags_used{$_} for keys %flags_used;
  exit 1   if $opt{VERSION} ; # Already showed version info  
  if ($opt{HELP}){
    print $HELP_TEXT;
    print "Program Options:\n",
         map {my ($h,$d); defined($h=$option_specs{$_}{HELP}) ?
                             "\  --$_ :$h" . (defined($d=$option_specs{$_}{DEFAULT}) ? " (default:$d)\n" : "\n")
							 : ""} sort keys %option_specs;
    exit 1;
  }  

  if (@ARGV){
    die "ERROR: Unknown argument (flag expected) : @ARGV";	  
  }
  if (-f $opt{FLAGFILE}){
    $opt{DEBUG} and print "--DEBUG: Reading Flagfile $opt{FLAGFILE}\n";
    open my $ff, "<", $opt{FLAGFILE} or die "ERROR: Could not open $opt{FLAGFILE}:$!";
    chomp (my @flag_options = grep !m/^\s*#/, <$ff>);
    close $ff;
    $opt{DEBUG} and print "--DEBUG: Flagfile option:'$_'\n;" for @flag_options;
    my %flagfile_option_value;
    Getopt::Long::GetOptionsFromArray(\@flag_options,\%flagfile_option_value, 
	                                  map {my $t; defined($t=$option_specs{$_}{TYPE}) ?
                                        ("$_$t"):()} keys %option_specs)
        or die "ERROR: Bad FlagFile $opt{FLAGFILE} Option\n";
    for my $k (keys %flagfile_option_value){
        if (exists $flags_used{$k}){  # Cmd line overrides flagfile
            $opt{DEBUG} and print "--DEBUG: Flagfile option $k=$flagfile_option_value{$k} IGNORED.. overridden by cmd line.\n";
        }elsif ($k eq "FLAGFILE"){
            die "ERROR: Nested flag files are not allowed."; 
        }else{
            $opt{DEBUG} and print "--DEBUG: Flagfile option $k=$flagfile_option_value{$k} set.\n";
            $opt{$k} = $flagfile_option_value{$k};
        }
    }
  }
  $opt{DEBUG} and print "DEBUG: $_\t",defined( $opt{$_})?$opt{$_}:"undef","\n" for sort keys %opt;
  my ($run_digits,$run_unit) = $opt{RUN_FOR} =~m/^(\d+)([dhms]?)$/i;
  $run_digits or die "ERROR:'RUN_FOR' option incorrectly specified($opt{RUN_FOR}). Use: 1d 3h 30s or just a number of seconds";
  $run_unit ||= "s"; # Default to seconds  
  my %unit_idx= (d=>24*3600,  m => 60 , h => 3600 ,s => 1); 
  $unit_idx{uc $_} = $unit_idx{$_} for keys %unit_idx;
  $opt{ENDTIME_EPOCH} = time() + $run_digits * $unit_idx{$run_unit};

  if ($opt{ANALYZE}){
	  return; # no more initialization needed   
  }

  $YBA_API = Web::Interface::->new();

  # Get universe name ..
  $opt{UNIVERSE} = $YBA_API->Get("");

  $opt{DEBUG} and print "--DEBUG:UNIV: $_\t","=>",$opt{UNIVERSE}{$_},"\n" for qw|name creationDate universeUUID version |;
  #my ($universe_name) =  $json_string =~m/,"name":"([^"]+)"/;
  if ($opt{UNIVERSE}{name}){
	 print "--UNIVERSE: ", $opt{UNIVERSE}{name}," on ", $opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{providerType}, " ver ",$opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{ybSoftwareVersion},"\n";
  }else{
     die "ERROR: Universe info not found \n";
  }
  $opt{NODES} = Analysis::Mode::Extract_nodes_From_Universe($opt{UNIVERSE}); 
  $opt{OUTPUT} ||= "queries." . unixtime_to_printable(time(),"YYYY-MM-DD") . ".$opt{UNIVERSE}{name}.mime.gz",
  $output = MIME::Write::Simple::->new(UNIVERSE_JSON=>$YBA_API->{json_string}); # No I/O so far 
  $output->Write_event(time(),"Querymonitor $VERSION Started for universe $opt{UNIVERSE}{name}",1); # Delayed write 
  my $nodestatus = $YBA_API->Get("/status");
  for my $nodename(keys %$nodestatus){
	next unless ref($nodestatus->{$nodename}) eq "HASH";
	next if $nodestatus->{$nodename}{node_status} eq "Live" 
	      and ($nodestatus->{$nodename}{master_alive} or $nodestatus->{$nodename}{tserver_alive});
	print "--WARNING: NODE $nodename Status:$nodestatus->{$nodename}{node_status}; ",
	       "Master-alive:$nodestatus->{$nodename}{master_alive}, Tserver-alive:$nodestatus->{$nodename}{tserver_alive}\n";
  }
  if ($opt{USETESTDATA}){
	 print "--Writing ONE record of test data, then exiting...\n";
	 $output->WriteQuery(time(), "ycql", {FIVE=>5,SIX=>6,COW=>"Moo"},"SELECT some_junk FROM made_up");
	 $output->Close("FINAL");
	 exit 2;
  }  
  # Run iteration ONCE, to verify it works...
  return unless $opt{DAEMON} ;
  
  $opt{LOCKFILE} .= ".$opt{UNIV_UUID}"; # one lock per universe 
  print "--Testing main loop before daemonizing...\n";

  if (Main_loop_Iteration()){
	  print "ERROR in main loop iteration. quitting...\n";
     exit 2;
  }
  # Close open file handles that may be leftover from main loop outputs
  $output->Close(0); # Not the "Final" close 
  print "--End main loop test.\n";  
}
#------------------------------------------------------------------------------
#------------------------------------------------------------------------------
sub daemonize {
    if ( ! $opt{DAEMON}){
      # Non-daemon mode -do this ONE time only ...
      warn "NOTE: Non-daemon mode is intended for TEST/DEBUG only\n";
      return $$;
    }
    my $grandchild_output = "nohup.out";
    sysopen $opt{LOCK_FH}, $opt{LOCKFILE}, O_EXCL | O_RDWR | O_CREAT | O_NONBLOCK
       or die "ERROR (Fatal):$0 is already running: Lockfile $opt{LOCKFILE}:$!";
     # Handle Signals ..
    sub Signal_Handler {   # 1st argument is signal name
        my($sig) = @_;
        #$sig ||= $! ||= "*Unknown*"; # Could  happen if sig within a sig ..
        my $msg =  "Caught a SIG=" . (defined($sig)? $sig:$!) . " --shutting down\n";
        warn $msg;
        close $opt{LOCK_FH}; # Note: LOCK, not Log - closing Since we will quit ...
        unlink $opt{LOCKFILE};
        $opt{LOCK_FH} = undef;
        $quit_daemon = 1; # Tell loop to quit
    }
    $SIG{$_}  = \&Signal_Handler 
        for qw{ INT KILL QUIT SEGV STOP TERM TSTP __DIE__}; # HUP
    $SIG{USR1}=sub{ warn( "Caught signal USR1. Setting DEBUG=" . ($opt{DEBUG} = (1 - $opt{DEBUG}))) 
                    # This sig will interrupt SLEEP, so we should automatically get another cycle.
                };
    my $grandpop_pid = $$; # Used to connect grandkid's pid message.
    #parent process to start another process that will go on its own is to "double fork."
    # 	The child itself forks and it then exits right away, so its child is taken over by init and can't be a zombie.
    warn (unixtime_to_printable(time()) 
	     . " Daemonizing. Expected to run in background until "
		 .  unixtime_to_printable($opt{ENDTIME_EPOCH}). "\n");
    my $pid = fork ();
    if ($pid < 0) {
      die "first fork failed: $!";
    } elsif ($pid) {
	  # This is the parent (grandpop) process. We want to exit() ASAP.
	  sleep 1; # Wait for grandkid to  write nohup.out 
	  exit 0 unless -f $grandchild_output;
	  open my $f, "<", $grandchild_output or exit 0;
	  my ($grandkid_pid) = grep {m/process (\d+) started\(from $grandpop_pid/} <$f>; 
	  close $f;
	  $grandkid_pid or exit 0;
	  ($grandkid_pid) = $grandkid_pid =~ m/process (\d+) started\(from $grandpop_pid/;
	  $grandkid_pid or exit 0;
      warn "To terminate daemon process, enter:\n"
            ."     kill -s QUIT $grandkid_pid\n"
            ."                              ('kill -s USR1 $grandkid_pid'  toggles DEBUG)\n";

      exit 0; # Exit the parent
    }
    # We are now a child process
    #close std fds inherited from parent
	close STDIN;
	close STDOUT;
	close STDERR;
	POSIX::setsid or die "setsid: $!"; # detaches future kids from the controlling terminal
    open (STDIN, "</dev/null"); # Null all file descriptors
    open (STDOUT, ">>$grandchild_output");
    open (STDERR, ">&STDOUT");
    my $grandchild = fork();
    if ($grandchild < 0) {
        die "second  fork failed : $!";
    } elsif ($grandchild) {
	    #print "Running as Daemon PID=$$\n";
		$| = 1;   # set autoflush on
	    print "-- ",unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS")," querymonitor child process $grandchild started(from $grandpop_pid). exiting parent(s)\n";
        exit 0; # Exit child, leaving grandkid running 		
    }
   ## chdir "/"; 
   POSIX::setsid or die "setsid: $!"; # detaches future kids from the controlling terminal
   umask 0; # Clear the file creation mask
   #foreach (0 .. (POSIX::sysconf (&POSIX::_SC_OPEN_MAX) || 1024))
   #   { POSIX::close $_ } # Close all open file descriptors

   return $pid; # Will always return "0", since this is the child process.
 }
#------------------------------------------------------------------------------
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
	die "ERROR: Unsupported format:'$format' ";
}
#------------------------------------------------------------------------------

#==============================================================================
BEGIN{
package Analysis::Mode; 

sub new{ # Unused
    my ($class) = @_;
    my $self = bless {}, $class;	
	$self->{INPUT}  = MIME::Multipart::ParseSimple::->new();
	$self->{PIECENBR} = 0;
	$self->{HEADER_PRINTED} =[0,0,0,0]; # Index by piece number 
	return $self;
}

sub Extract_nodes_From_Universe{ # CLASS method 
    my ($univ_hash, $callback) = @_;
	
	my @node;
	my $count=0;
	for my $n (@{  $univ_hash->{universeDetails}{nodeDetailsSet} }){
       push @node, my $thisnode = {map({$_=>$n->{$_}||''} qw|nodeIdx nodeName nodeUuid azUuid isMaster
                                  	   isTserver ysqlServerHttpPort yqlServerHttpPort state|),
	                              map({$_=>$n->{cloudInfo}{$_}} qw|private_ip public_ip az region |) };
       $callback and $callback->($thisnode, $count);
       $count++;
    }
    return [@node];	
}

sub Handle_MIME_HEADER  {
	my ($self,$dispatch_type) = @_;
	
	return unless $dispatch_type eq "Header"; # Only expecting a Header 

    for ( grep {!/_SECTION_|Content-Type|params$/} sort keys %{$self->{INPUT}{general_header}} ){
		   print {$self->{OUTPUT_FH}} "INSERT INTO kv_store VALUES ('MIMEHDR','$_','$self->{INPUT}{general_header}{$_}');\n";
	}
		
}

sub Handle_UNIVERSE_JSON{
	my ($self,$dispatch_type, $body) = @_;
	$opt{DEBUG} and print "--DEBUG: IN: UNIV JSON handler type $dispatch_type\n";
    return unless $dispatch_type eq "Body"; # Only Interested in Body 
    if (length($body) < 10){
       print "--ERROR: UNIVERSE Info was not found in '$body'\n";
       return;
    }
    my $bj = JSON::Tiny::decode_json($body);

	$self->{node} = Extract_nodes_From_Universe($bj,
	   sub{my ($n,$count) = @_;
		  if ($count== 0){
		     $opt{DEBUG} and print "--DEBUG:","CREATE TABLE NODE (",
			      join(",", sort keys %$n ), ");\n";
		     print {$self->{OUTPUT_FH}} "CREATE TABLE IF NOT EXISTS NODE (",
			      join(",", sort keys %$n ), ");\n";
		  }
		  $opt{DEBUG} and print "--DEBUG:","INSERT INTO NODE VALUES('",
			      join("','", map{$n->{$_}} sort keys %$n ), "');\n";
		  print {$self->{OUTPUT_FH}} "INSERT INTO NODE VALUES('",
			      join("','", map{$n->{$_}} sort keys %$n ), "');\n";
		  $self->{REGION}{ $n->{region} }{NODE_COUNT}++; # Track available regions 	   
	   });
    print {$self->{OUTPUT_FH}} "----- End of nodes -----\n";
	for my $k (qw|universeName provider providerType replicationFactor numNodes ybSoftwareVersion enableYCQL
             	enableYSQL enableYEDIS nodePrefix |){
	   next unless defined ( my $v= $bj->{universeDetails}{clusters}[0]{userIntent}{$k} );
	   print {$self->{OUTPUT_FH}} "INSERT INTO kv_store VALUES ('CLUSTER','$k','$v');\n";
	}
	for my $flagtype (qw|masterGFlags tserverGFlags |){
	   next unless my $flag = $bj->{universeDetails}{clusters}[0]{userIntent}{$flagtype};
	   for my $k(sort keys %$flag){
		  my $v = $flag->{$k}; 
	      print {$self->{OUTPUT_FH}} "INSERT INTO kv_store VALUES ('$flagtype','$k','$v');\n";
	   }
	}
	for my $region (@{ $bj->{universeDetails} {clusters} [0]{placementInfo}{cloudList}[0]{regionList} }){
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
}

sub Handle_CSVHEADER	{
	my ($self,$dispatch_type) = @_;
	$opt{DEBUG} and print "--DEBUG:IN:CSVHEADER handler type $dispatch_type\n";
	return unless $dispatch_type eq "Header";
	my $type = $self->{INPUT}{general_header}{TYPE};
	my ($Ignore_type_field_in_fieldnames,$timestamp_field,@field_names) 
	           = split /,/, $self->{INPUT}{general_header}{FIELDS};
    $self->{FIELDS}{$type} = [@field_names];
	$_ = "$_ INTEGER" for grep {/milli|_ms/i} @field_names; # Make MILLISECONDS an integer 
	$opt{DEBUG} and print "--DEBUG:","CREATE TABLE IF NOT EXISTS $type ($timestamp_field INTEGER,",
                    join(",",@field_names), ");\n";
	print {$self->{OUTPUT_FH}} "CREATE TABLE IF NOT EXISTS $type ($timestamp_field INTEGER,",
	                 join(",",@field_names), ");\n";
	$self->{TYPE_EXISTS}{$type}++;
}
sub Handle_MONITOR_DATA {
	my ($self,$dispatch_type,$body ) = @_;
	$opt{DEBUG} and print "--DEBUG:IN: MonitorData handler type $dispatch_type\n";
	return unless $dispatch_type eq "Body" and length($body) > 2;
	my @values = ();  # Poor man's Text::CSV, from Perl FAQ. no embedded " allowed.
    push(@values, $+) while $body =~ m{
			"([^\"\\]*(?:\\.[^\"\\]*)*)",? # groups the phrase inside the quotes
		   | ([^,]+),?
		   | ,
		}gx;
	for (@values){
		$_ = '' unless defined $_;
	    s/'/~/g ;
	}
	my $type = shift @values;
	print {$self->{OUTPUT_FH}} "INSERT INTO $type VALUES('",
	        join("','",@values),      "');\n"; 
};

sub Handle_Event_Data{
	my ($self,$dispatch_type,$body ) = @_;
	$opt{DEBUG} and print "--DEBUG:IN: EVENT handler type $dispatch_type\n";
	return unless $dispatch_type eq "Header";
	# Put stuff into EVENT table 
	for my $ts (sort keys %{  $self->{INPUT}{general_header} }){
	  print {$self->{OUTPUT_FH}} "INSERT INTO event VALUES($ts,'",
	        $self->{INPUT}{general_header}{$ts}  ,
	        "');\n"; 
	}
}

my %Section_Handler =( # Defines Handler subroutines for each Mime piect (_SECTION_) received
	MIME_HEADER		=> \&Handle_MIME_HEADER	,
	UNIVERSE_JSON	=> \&Handle_UNIVERSE_JSON ,
	CSVHEADER		=> \&Handle_CSVHEADER	,
	MONITOR_DATA	=> \&Handle_MONITOR_DATA,
	EVENT           => \&Handle_Event_Data,
);

sub Parse_Body_Record{
   my ($self, $rec) = @_;

   my $dispatch = $Section_Handler{ $self->{INPUT}{general_header}{_SECTION_} };
   if ( ! $self->{HEADER_PRINTED}[$self->{PIECENBR}]){
      $opt{DEBUG} and print "--DEBUG:HDR $self->{PIECENBR}:",map({"$_=>" . $self->{INPUT}{general_header}{$_} . "; "} 
	     grep {!/params$/} sort keys %{$self->{INPUT}{general_header}}),"\n";
	  print {$self->{OUTPUT_FH}} "BEGIN TRANSACTION; -- $self->{PIECENBR} : ",
	          $self->{INPUT}{general_header}{_SECTION_}, "\n";
	  $dispatch and $dispatch->($self,"Header"); # Handler is called here 
	  $self->{HEADER_PRINTED}[$self->{PIECENBR}] = 1;
   }

   if ( ! defined $rec) {
      $opt{DEBUG} and print "--DEBUG:---- PIECE COMPLETE --\n" ;
	  $dispatch and $dispatch->($self,"EOF"); # Handler is called here 
	  print {$self->{OUTPUT_FH}} "END TRANSACTION; -- $self->{PIECENBR} : ",
	          $self->{INPUT}{general_header}{_SECTION_}, "\n";
	  $self->{PIECENBR}++;
	  return;
   }   
   $opt{DEBUG} and print "--DEBUG:GOT Piece:",substr($rec,0,200),"..\n";
   $dispatch and $dispatch->($self,"Body", $rec); # Handler is called here    
}

sub Process_file_and_create_Sqlite{
	my ($self) = @_;
	
	if ( -f $opt{ANALYZE} ){
		# File exists - fall through and process it
	}else{
	    die "ERROR: 'ANALYZE' file does not exist: No file '$opt{ANALYZE}'.";	
	}
	print "--Analyzing $opt{ANALYZE} ...\n";
	# Create INPUT handle to read (possibly gzipped) MIME file ---
	if ($opt{ANALYZE} =~/\.gz$/i){
		open($self->{INPUT_FH}, "-|", "gunzip -c $opt{ANALYZE}") or die "ERROR: Cannot fork gunzip to open $opt{ANALYZE}";
	}else{
		open($self->{INPUT_FH},"<",$opt{ANALYZE}) or die "ERROR opening $opt{ANALYZE}:$!";
	}
	
    $self->Initialize_SQLITE_Output();
	
	$self->{INPUT}->parse($self, \&Parse_Body_Record);
	close $self->{INPUT_FH};
	print {$self->{OUTPUT_FH}} "SELECT 'All input records processed.';\n";

	$self->{TYPE_EXISTS}{ycql} and $self->Create_and_run_views_for_ycql();
	$self->{TYPE_EXISTS}{ysql} and $self->Create_and_run_views_for_ysql();
	close $self->{OUTPUT_FH};
	print "--For detailed analysis, run: $opt{SQLITE} -header -column $self->{SQLITE_FILENAME}\n";
	return;
}

sub Create_and_run_views_for_ycql{
    my ($self) = @_;
	
	my ($count_by_region_and_type,$count_by_region) = ("","");
	for my $r (sort keys %{ $self->{REGION} }){
	   my $preferred =  $self->{REGION}{$r}{PREFERRED} ? "(P)" : "";
	   $count_by_region_and_type .= "sum(case when instr(query,' system.')>0 and region='$r' then 1 else 0 end) as [sys_$r],\n";
	   $count_by_region_and_type .= "sum(case when instr(query,' system.')=0 and region='$r' then 1 else 0 end) as [cql_$r$preferred],\n";
	   $count_by_region    .= "sum ( CASE WHEN region = '$r' THEN 1 ELSE 0 END) as [${r}_queries],\n";
	}
	$count_by_region=~s/,$//; # Zap trailing comma 
   my ($elapsed_ms) = grep {m/milli/i} @{ $self->{FIELDS}{ycql} }; 	
   print {$self->{OUTPUT_FH}} <<"__Summary_SQL__";
CREATE VIEW IF NOT EXISTS summary_cql as
SELECT datetime((ts/600)*600,'unixepoch') as UTC,
    sum(case when instr(query,' system.')> 0 then 1 else 0 end) as systemq,
        sum(case when instr(query,' system.')=0 then 1 else 0 end) as cqlcount,
        sum(case when instr(query,' system.')>0 and $elapsed_ms > 120 then 1 else 0 end) as sys_gt120,
        sum(case when instr(query,' system.')=0 and $elapsed_ms > 120 then 1 else 0 end) as cql_gt120,
        $count_by_region_and_type
        round(sum(case when instr(query,' system.')=0 and $elapsed_ms > 120 then 1 else 0 end) *100.0
               / sum(case when instr(query,' system.')=0 then 1 else 0 end)
                   ,2) as breach_pct
FROM ycql,node 
WHERE ycql.nodename=node.nodename  
GROUP BY UTC;

CREATE VIEW IF NOT EXISTS slow_queries AS
SELECT query, count(*) as nbr_querys, round(avg($elapsed_ms),1) as avg_milli ,
      sum (CASE when $elapsed_ms > 120 then 1 else 0 END)*100 / count(*) as pct_gt120,
      $count_by_region
FROM ycql, node
WHERE ycql.nodename=node.nodename
GROUP BY query
HAVING nbr_querys > 50 and avg_milli >10  ORDER by avg_milli  desc;

CREATE VIEW IF NOT EXISTS node_summary_cql AS 
SELECT ycql.nodename, round(avg($elapsed_ms),1) as avg_ms, 
       count(*), sum(case when instr(query,' system.') > 0 then 1 else 0 end) as sys_count,  
	   sum(case when instr(query,' system.')= 0 then 1 else 0 end) as cql_count,
	   sum(case when instr(query,' system.')= 0 and $elapsed_ms > 120 then 1 else 0 end) as cql_gt_120,
	   sum(case when instr(query,' system.')> 0 and $elapsed_ms > 120 then 1 else 0 end) as sys_gt_120,
	   region
FROM ycql,node 
WHERE ycql.nodename=node.nodename   
GROUP by  ycql.nodename 
ORDER by ycql.nodename;

CREATE VIEW IF NOT EXISTS q_detail AS 
SELECT ts,DATETIME((ts),'unixepoch') as UTC, 
  $elapsed_ms, ycql.nodeName, "query", 
  substr(remote_ip,1,instr(remote_ip,':')-1) as client,
  "type",
  upper(substr(ltrim(query,' '),1,6)) as verb,
  CASE WHEN upper(substr(ltrim(query,' '),1,6))='UPDATE' 
       then substr(ltrim(query,' '),8,18)
       ELSE substr(query,instr(lower(query),' from ')+6,18) 
   END as tbl,
   region
FROM ycql,node 
WHERE ycql.nodename=node.nodename  
      AND instr(query,' system.')=0 
;
CREATE VIEW IF NOT EXISTS client_summary AS 
SELECT client, type,verb, round(avg($elapsed_ms),0) as avg_millis,
       $count_by_region
 from  q_detail 
 group by client,type,verb;

CREATE VIEW IF NOT EXISTS slow_tables AS
SELECT tbl,type,verb,count(*) as queries,
         round(avg($elapsed_ms),0) as avg_millis,
         sum (CASE when $elapsed_ms > 120 then 1 else 0 END)*100 / count(*) as pct_gt120
FROM q_detail 
GROUP BY tbl,type,verb
ORDER BY avg_millis*queries  desc 
LIMIT 25;
 
.header off
.mode column
SELECT 'Imported ' || count(*) ||' ycql rows from $opt{ANALYZE}.' as Imported_count from ycql;
SELECT '====== summary_cql Report ====';
.header on
SELECT * from summary_cql;
.header off
SELECT '';
SELECT '======= Slow Queries =======';
.header on
select * from slow_queries;
__Summary_SQL__
}

sub Create_and_run_views_for_ysql{
    my ($self) = @_;
	print {$self->{OUTPUT_FH}} 
	  "SELECT 'Imported ' || count(*) ||' ysql rows from $opt{ANALYZE}.' as Imported_count from ysql;\n";
}
sub Initialize_SQLITE_Output{
    my ($self) = @_;
	$!=undef;
	if ($opt{OUTPUT}){
		if ($opt{OUTPUT} =~/^STDOUT$/i){ # Send SQL to stdout
			die "ERROR: Cannot specify DB and OUTPUT=STDOUT together\n" if $opt{DB};
			$opt{DB} = " ";
			$opt{SQLITE} = "cat";
		}else{
			die "ERROR: The only valid OUTPUT option is STDOUT in --analyze mode. use --DB.\n";
		}
	}
	my ($sqlite_version) = $opt{SQLITE} ? do {my $vv=qx|$opt{SQLITE} --version|;chomp $vv;$vv=~/([\d\.]+)/}
	                       : ("N/A");
	$! and die "ERROR: Cannot run $opt{SQLITE} :$!";
	# Create output handle to SQLITE ----
	($self->{SQLITE_FILENAME}) = $opt{ANALYZE} =~m/(.+)\.\w+$/;
	$self->{SQLITE_FILENAME}   =~s/\.(?:csv|mime)$//i; # Zap the mime or csv 
	$self->{SQLITE_FILENAME}   .= ".sqlite"; # Add sqlite suffix
	$opt{DB}  and    $self->{SQLITE_FILENAME} = $opt{DB}; # DB wins, if specified.
	-e $self->{SQLITE_FILENAME} and die "ERROR: $self->{SQLITE_FILENAME} already exists. Use --db to specify a different file.\n";
	print "--Populating Sqlite($sqlite_version) database '$self->{SQLITE_FILENAME}'...\n";
	
	open($self->{OUTPUT_FH}, "|-", "$opt{SQLITE} $self->{SQLITE_FILENAME}") 
	     or die "ERROR: Cannot fork sqlite to open $self->{SQLITE_FILENAME}";
	print {$self->{OUTPUT_FH}} <<"__SQL1__";
CREATE TABLE IF NOT EXISTS kv_store(type text,key text, value text);
INSERT INTO kv_store VALUES 
	   ('GENERAL','Analysis_host','$opt{HOSTNAME}')
	  ,('GENERAL','Analysis date','$opt{STARTTIME_TZ}')
	  ,('GENERAL','Processing file','$opt{ANALYZE}')
	  ,('GENERAL','Analysis version','$main::VERSION');
CREATE TABLE IF NOT EXISTS event(ts INTEGER, e TEXT);
__SQL1__

}

1;
} # End of Analysis::Mode
#==============================================================================
BEGIN{
package Web::Interface; # Handles communication with YBA API 

sub new{
	my ($class) = @_;
	for(qw|API_TOKEN YBA_HOST CUST_UUID UNIV_UUID|){
        $opt{$_} or die "ERROR: Required parameter --$_ was not specified.\n";
    }
	for(qw|API_TOKEN  CUST_UUID UNIV_UUID|){
        (my $value=$opt{$_})=~tr/-//d; # Extract and zap dashes
		my $len = length($value);
		next if $len == 32; # Expecting these to be exactly 32 bytes 
        warn "WARNING: Expecting 32 valid bytes in Option $_=$opt{$_} but found $len bytes. \n";
		sleep 2;
    }	
	my $self =bless {map {$_ => $opt{$_}} qw|HTTPCONNECT UNIV_UUID API_TOKEN YBA_HOST CUST_UUID| }, $class;
	my $http_prefix = $opt{YBA_HOST} =~m/^http/i ? "" : "HTTP://";
    $self->{BASE_URL_API_CUSTOMER} = "${http_prefix}$opt{YBA_HOST}/api/customers/$opt{CUST_UUID}/universes/$opt{UNIV_UUID}";
	$self->{BASE_URL_UNIVERSE}     = "${http_prefix}$opt{YBA_HOST}/universes/$opt{UNIV_UUID}";
	$http_prefix ||= substr($opt{YBA_HOST},0,5); # HTTP: or HTTPS
	if ($self->{HTTPCONNECT} eq "curl"){
		  $self->{curl_base_cmd} = join " ", $opt{CURL}, 
					 qq|-kfs --request GET --header 'Content-Type: application/json'|,
					 qq|--header "X-AUTH-YW-API-TOKEN: $opt{API_TOKEN}"|;
		  if ($opt{DEBUG}){
			 print "--DEBUG:CURL base CMD: $self->{curl_base_cmd}\n";
		  }
		  return $self;
    }
    if ($http_prefix =~/^https/i){
	   my ($ok, $why) = HTTP::Tiny->can_ssl();
       if (not $ok){
          print "ERROR: HTTPS requested , but perl modules are insufficient:\n$why\n";
		  print "You can avoid this error, if you use the (less efficient) '--HTTPCONNECT=curl' option\n";
		  die "ERROR: HTTP::Tiny module dependencies not satisfied for HTTPS.";
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
	my ($self, $endpoint, $base) = @_;
	$self->{json_string}= "";
	my $url = $base ? $self->{$base} : $self->{BASE_URL_API_CUSTOMER};
	if ($self->{HTTPCONNECT} eq "curl"){
		$self->{json_string} = qx|$self->{curl_base_cmd} --url $url$endpoint|;
		if ($?){
		   print "ERROR: curl get '$endpoint' failed: $?\n";
		   exit 1;
		}
    }else{ # HTTP::Tiny
	   $self->{raw_response} = $self->{HT}->get(  $url . $endpoint );
	   if (not $self->{raw_response}->{success}){
		  print "ERROR: Get '$endpoint' failed with status=$self->{raw_response}->{status}: $self->{raw_response}->{reason}\n";
		  $self->{raw_response}->{status} == 599 and print "\t(599)Content:$self->{raw_response}{content};\n";
		  exit 1;
	   }
	   $self->{json_string} = $self->{raw_response}{content};
	}
	$self->{response} = JSON::Tiny::decode_json( $self->{json_string} );
	return $self->{response};
}

} # End of  package Web::Interface;  
#==============================================================================
BEGIN{
package MIME::Write::Simple;

sub new{
	my ($class,%att) = @_;
	
	$att{boundary} ||= time() . $$;
	$att{MIME_VERSION_SENT} = 0;
	return bless {%att},$class;
}

sub header{
	my ($self,$content_type,$header_msg) = @_;
	$header_msg ||= "";
	chomp $header_msg;
	$header_msg .= "\n\n";
	my $mime_ver = $self->{MIME_VERSION_SENT} ? "" 
                   : "MIME Version: 1.0\n";
    $self->{MIME_VERSION_SENT} = 1;
	print { $self->{OUTPUT_FH} }   $mime_ver
	      . qq|Content-Type: $content_type; boundary="$self->{boundary}"\n|
		  . $header_msg ;
}

sub boundary{ # getter/setter
   my ($self,$b, $final) = @_;
   $b and $self->{boundary} = $b;
   print { $self->{OUTPUT_FH} }  "--" . $self->{boundary} . ($final ? "--\n" : "\n");
}

sub Open_and_Initialize{
	my ($self) = @_;
	$opt{DEBUG} and print "--DEBUG: Opening output file=" , $opt{OUTPUT},"\n";
	my $output_already_exists = -f $opt{OUTPUT};
	open $self->{OUTPUT_FH} , "|-", "gzip -c >> " . $opt{OUTPUT}
	  or die "ERROR: Cannot fork output zip:$!";
	if (! $output_already_exists){
	   # Need MIME headers etc...
	   $self->header("multipart/mixed",
	      join ("\n",
			  "Querymonitor_version: $VERSION",
			  "UNIVERSE: $opt{UNIVERSE}{name}",
			  #"UNIV_UUID: $opt{UNIV_UUID}",
			  #"STARTTIME: $opt{STARTTIME_TZ}",
			  "Collection_host: $opt{HOSTNAME}",
			  "_SECTION_: MIME_HEADER",
			  map({"$_: $opt{$_}"} grep{defined $opt{$_} and ref $opt{$_} eq "" } sort keys %opt)
		  ));
	    $self->boundary();  
        # Insert NODE/ZONE info
        $self->header("application/json", 
                      "Description: Universe detail JSON\nName: universe_info\n"
					  ."_SECTION_: UNIVERSE_JSON");
		print { $self->{OUTPUT_FH} } $self->{UNIVERSE_JSON},"\n";
		$self->boundary();
	}
}

sub Write_event{
   my ($self,$ts,$event,$delayed_write) = @_;
   if ($delayed_write){
      push @{$self->{EVENT_QUEUE}}, [$ts,$event];
	  return;
   }
   return unless  @{$self->{EVENT_QUEUE}} or $event;
   if (! $self->{OUTPUT_FH} ){
	  # Need to initialize output
      $self->Open_and_Initialize();
   }   
   if ($self->{IN_CSV_SECTION}){
	   $self->{IN_CSV_SECTION} = 0;
	   $self->{TYPE_INITIALIZED} = {}; # Zap types to un-init 
	   $self->boundary(); # Close the CSV section
   }
 	$self->header("text/event",
	      join("\n",
			 "_SECTION_: EVENT",
			 map ({$_->[0] .": " . $_[1]}  @{$self->{EVENT_QUEUE}}),
			 $event ? ("$ts: $event") : ()
			 ));
	$self->boundary();  
	$self->{EVENT_QUEUE} = [];
}

sub Initialize_query_type{
	my ($self,$type,$q) = @_;
	
    if (! $self->{OUTPUT_FH} ){
	  # Need to initialize output
      $self->Open_and_Initialize();
    }
    if ($self->{IN_CSV_SECTION}){
	   $self->boundary(); # Close the CSV section
	}
	$self->header("text/csvheader",
	      join("\n",
		     "TYPE: $type",
			 "_SECTION_: CSVHEADER",
			 "FIELDS: " . join(",","type","ts",sort(keys %$q))
			 ));
	$self->boundary();
	$self->{TYPE_INITIALIZED}{$type} = 1;
	$self->header("text/csv","_SECTION_: MONITOR_DATA");
	$self->{IN_CSV_SECTION} = 1;
}

sub WriteQuery{
  my ($self,$ts, $type, $q, $sanitized_query) = @_;
  
  if ( ! $self->{TYPE_INITIALIZED}{$type} ){
	  $self->Initialize_query_type($type, $q);  
  }
  $sanitized_query =~tr/"\n/~ /; # Zap internal double quotes and \n - which will mess CSV
  if (length($sanitized_query) > $opt{MAX_QUERY_LEN}){
	   $sanitized_query = substr($sanitized_query,0,$opt{MAX_QUERY_LEN}/2 -2) 
	                     . ".." . substr($sanitized_query,-($opt{MAX_QUERY_LEN}/2));
  }
  $q->{query} = qq|"$sanitized_query"|;
  print { $self->{OUTPUT_FH} } join(",", $type, $ts, map( {$q->{$_}} sort keys %$q)),"\n";
}

sub Close{
	my ($self, $final) = @_;
    return unless $self->{OUTPUT_FH};
	$final and $self->Write_event(time(),"Final close",0); # No delay 
	$self->boundary(undef,$final);
	$self->{IN_CSV_SECTION} = 0;
    close $self->{OUTPUT_FH};
    $self->{OUTPUT_FH} = undef;
	$self->{TYPE_INITIALIZED} ={};
}	
1;
} # End of MIME::Write::Simple
#==============================================================================
BEGIN{
package MIME::Multipart::ParseSimple;

use strict;
use warnings FATAL => 'all';
use Carp;

our $VERSION = '0.02';


sub new {
  my $p = shift;
  my $c = ref($p) || $p;
  my $o = {};
  bless $o, $c;
  return $o;
}

=head2 parse

takes one argument: a file handle.

returns a listref, each item corresponding to a MIME header in
the document.  The first is the multipart file header itself.
Each header item is stored as key/value.  Additional parameters
are stored $key.params.  e.g. the boundary is at

    $o->[0]->{"Content-Type.params"}->{"boundary"}

The first item may also have {"Preamble"} and {"Epilog"} if these
existed in the file.

The content of each part is stored as {"Body"}.

=cut

sub parse {
  # load a MIME-multipart-style file containing at least one application/x-ptk.markdown
  my ($o,$caller, $callback) = @_;
  $o->{fh}       = $caller->{INPUT_FH};
  $o->{CALLER}   = $caller;
  $o->{callback} = $callback;

  my $mp1 = readline($o->{fh});
  my $mp1e = 'MIME Version: 1.0';
  die "Multipart header line 1 must begin ``$mp1e'' " unless $mp1 =~ /^$mp1e/;
 
  $o->{general_header} = $o->parseHeader();
  croak "no boundary defined" unless $o->{general_header}->{"Content-Type.params"}->{"boundary"};
  my $b = $o->{general_header}->{"Content-Type.params"}->{"boundary"};
  if (length($b)>2 and (my $quote=substr($b,0,1)) eq substr($b,-1,1)){
      if ($quote eq '"'  or $quote eq "'"){
		  $b=substr($b,1,length($b)-2);
	  }
  }
  $o->{boundary} = $b;
  
  $o->parseBody();

  #my @parts = ($general_header);

  while(! (eof($o->{fh}) || $o->{eof})){
    $o->{general_header} = $o->parseHeader();
    $o->parseBody();
    #push @parts, $header;
  }
  
  if (! $o->{eof}){
	 # Did not find proper ending boundary
	 $o->{callback}->($o->{CALLER},undef); # Indiates Piece completed.
  }
  #$general_header->{Epilog} = $o->parseBody();

  #return \@parts;

}

sub parseBody {
  my ($o) = @_;
  my $fh = $o->{fh};
  my $boundary = $o->{boundary};
  $o->{recordnumber} = 0;
  while(<$fh>){
    $o->{eof} = 1 if /^--$boundary--/;
    if (/^--$boundary/){
		$o->{callback}->($o->{CALLER},undef); # Indiates Piece completed.
		$o->{general_header} = undef;
		return;
	}
	$o->{recordnumber}++;
	chomp;
    $o->{callback}->($o->{CALLER},$_);
  }
  return undef;
}

sub parseHeader {
  my ($o) = @_;
  my $fh = $o->{fh};
  my %header = ();
  my ($k,$v,$e,$p);
  while(<$fh>){
    last if /^\s*$/; # break on a blank line...
    my @parts = split /;/;
    if(/^\S/){ # non space at start means a new header item
      my $header = shift @parts;
      ($k,$v) = split(/\:/, $header, 2);
	  next unless $v;
      $k =~ s/(?:^\s+|\s+$)//g;
      $v =~ s/(?:^\s+|\s+$)//g;
      $header{$k} = $v;
      $p = $k.'.params';
      $header{$p} = {};
    }
    foreach my $part(@parts){
      my ($l,$w) = split(/=/, $part, 2);
	  next unless $w;
      $l =~ s/(?:^\s+|\s+$)//g;
      $w =~ s/(?:^\s+|\s+$)//g;
      $header{$p}->{$l} = $w;
    }
  }
  return \%header;
}
1;
} # End of  MIME::Multipart::ParseSimple
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