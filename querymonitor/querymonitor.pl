#!/usr/bin/perl

our $VERSION = "0.08";
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

my %opt=(
    API_TOKEN                      => $ENV{API_TOKEN},
    YBA_HOST                       => $ENV{YBA_HOST},
    CUST_UUID                      => $ENV{CUST_UUID},
    UNIV_UUID                      => $ENV{UNIV_UUID},

    # Operating variables
    STARTTIME	                  => unixtime_to_printable(time(),"YYYY-MM-DD HH:MM"),
    INTERVAL_SEC                  => 5,             # You can put fractions of a second here
    RUN_FOR                       => "4h",  # 4 hours
    ENDTIME_EPOCH                 => 0, # Calculated after options are processed
    CURL                          => "curl",
    FLAGFILE                      => "querymonitor.defaultflags",
	YSQL_OUTPUT                   => "queries.ysql." . unixtime_to_printable(time(),"YYYY-MM-DD") . ".csv.gz",
	YCQL_OUTPUT                   => "queries.ycql." . unixtime_to_printable(time(),"YYYY-MM-DD") . ".csv.gz",
    # Misc
    DEBUG                         => 0,
    HELP                          => 0,
    DAEMON                        => 1,
    LOCKFILE                      => "/var/lock/querymonitor.lock",
    LOCK_FH                      => undef,
	MAX_QUERY_LEN                => 2048,
	SANITIZE					 => 0,   # Remove PII 
	ANALYZE						 => undef,     # Input File-name ( "..csv.gz" ) to process through sqlite
	DB                           => undef,     # output SQLITE database file name
	SQLITE                       => "sqlite3", # path to Sqlite binary
);

my $quit_daemon = 0;
my $loop_count  = 0;
my $error_counter=0;
my $curl_cmd; # Populated in `Initialize`
my %outinfo; # For handles, and headers 

Initialize();

if ($opt{ANALYZE}){
   Process_the_CSV_through_Sqlite();
   exit 0;
}

daemonize();

#------------- M a i n    L o o p --------------------------
while (not ($quit_daemon  or  time() > $opt{ENDTIME_EPOCH} )){  # Infinite loop ...
   $loop_count++;
   Main_loop_Iteration();
   sleep($opt{INTERVAL_SEC});
}
#------------- E n d  M a i n    L o o p ---------------
# Could get here if a SIGNAL is received
warn(unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS") ." Program $$ Completed after $loop_count iterations.\n"); 

$opt{LOCK_FH} and close $opt{LOCK_FH} ;  # Should already be closed and removed by sig handler
unlink $opt{LOCKFILE};
for (keys %outinfo){
	close $outinfo{$_}{FH};
}

exit 0;
#------------------------------------------------------------------------------
#==============================================================================
#------------------------------------------------------------------------------
sub Main_loop_Iteration{

    my $query_type = "Unknown";

    if ($opt{DEBUG}){
        print "DEBUG: Start main loop iteration $loop_count\n";
    }
    my $json_string = qx|$curl_cmd|;
    if ($?){
        print "ERROR: curl command failed: $?\n";
        $error_counter++;
        return 1;
    }
	my $ts = time();
    $opt{DEBUG} and print "DEBUG:$loop_count:$json_string\n";
	$json_string !~/{/ and die "ERROR: $json_string";
    $json_string =~s/":/"=>/g; # Convert to perl structure
    my $queries = eval $json_string;
    $@ and die "ERROR: Could not eval json:Eval err $@";
    
    for my $type (qw|ysql ycql|){
       for my $q (@{ $queries->{$type}{queries} }){
		   for my $subquery (split /;/, $q->{query}){
			 #Sanitize PII from each
			 my $sanitized_query = $opt{SANITIZE} ? SQL_Sanitize($subquery) : $subquery;
             #print join(",",$ts, $type,$q->{nodeName},$q->{id},$subquery),"\n";
			 Output_Handler($ts, $type, $q, $sanitized_query);
		   }
       }
    }

  return 0;
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
sub Output_Handler{
  my ($ts, $type, $q, $sanitized_query) = @_;
  
  $sanitized_query =~tr/"/~/; # Zap internal double quotes - which will mess CSV
  if (length($sanitized_query) > $opt{MAX_QUERY_LEN}){
	   $sanitized_query = substr($sanitized_query,0,$opt{MAX_QUERY_LEN}/2 -2) 
	                     . ".." . substr($sanitized_query,-($opt{MAX_QUERY_LEN}/2));
  }
  $q->{query} = qq|"$sanitized_query"|;
  if (! $outinfo{$type} ){
	  $opt{DEBUG} and print "DEBUG: Opening ($type) output file=" , $opt{uc $type . "_OUTPUT"},"\n";
	  my $output_already_exists = -f $opt{uc $type . "_OUTPUT"};
	  open $outinfo{$type}{FH} , "|-", "gzip -c >> " . $opt{uc $type . "_OUTPUT"}
     	  or die "ERROR: Cannot fork for $type output zip:$!";
	  if (! $output_already_exists){
		  print { $outinfo{$type}{FH} } join(",","ts",sort keys %$q),"\n";
	  }
  }
  print { $outinfo{$type}{FH} } join(",", $ts, map( {$q->{$_}} sort keys %$q)),"\n";
}

#------------------------------------------------------------------------------
sub Initialize{
  chomp ($opt{HOSTNAME} = qx|hostname|);
  print $opt{STARTTIME}," Starting $0 version $VERSION  PID $$ on $opt{HOSTNAME}\n";

  my @program_options = qw[ DEBUG! HELP! DAEMON! SANITIZE!
                       API_TOKEN=s YBA_HOST=s CUST_UUID=s UNIV_UUID=s
                       INTERVAL_SEC=i RUN_FOR|RUNFOR=s CURL=s SQLITE=s
                       FLAGFILE=s YSQL_OUTPUT=s YCQL_OUTPUT=s DB=s
					   MAX_QUERY_LEN|MAXQL|MQL=i ANALYZE|PROCESS=s];
  my %flags_used;
  Getopt::Long::GetOptions (\%flags_used, @program_options)
      or die "ERROR: Bad Option\n";
  $opt{$_} = $flags_used{$_} for keys %flags_used; # Apply cmd-line flags immediately
  if ($opt{HELP}){
    print $HELP_TEXT;
    print "Program Options:\n",
         map {my $x=$_; $x=~s/\W.*//g; "\t--$x\n"} sort @program_options;
    exit 1;
  }
  if (-f $opt{FLAGFILE}){
    $opt{DEBUG} and print "DEBUG: Reading Flagfile $opt{FLAGFILE}\n";
    open my $ff, "<", $opt{FLAGFILE} or die "ERROR: Could not open $opt{FLAGFILE}:$!";
    chomp (my @flag_options = grep !m/^\s*#/, <$ff>);
    close $ff;
    $opt{DEBUG} and print "DEBUG: Flagfile option:'$_'\n;" for @flag_options;
    my %flagfile_option_value;
    Getopt::Long::GetOptionsFromArray(\@flag_options,\%flagfile_option_value, @program_options)
        or die "ERROR: Bad FlagFile $opt{FLAGFILE} Option\n";
    for my $k (keys %flagfile_option_value){
        if (exists $flags_used{$k}){  # Cmd line overrides flagfile
            $opt{DEBUG} and print "DEBUG: Flagfile option $k=$flagfile_option_value{$k} IGNORED.. overridden by cmd line.\n";
        }elsif ($k eq "FLAGFILE"){
            die "ERROR: Nested flag files are not allowed."; 
        }else{
            $opt{DEBUG} and print "DEBUG: Flagfile option $k=$flagfile_option_value{$k} set.\n";
            $opt{$k} = $flagfile_option_value{$k};
        }
    }
  }

  my ($run_digits,$run_unit) = $opt{RUN_FOR} =~m/^(\d+)([dhms]?)$/i;
  $run_digits or die "ERROR:'RUN_FOR' option incorrectly specified($opt{RUN_FOR}). Use: 1d 3h 30s or just a number of seconds";
  $run_unit ||= "s"; # Default to seconds  
  my %unit_idx= (d=>24*3600,  m => 60 , h => 3600 ,s => 1); 
  $unit_idx{uc $_} = $unit_idx{$_} for keys %unit_idx;
  $opt{ENDTIME_EPOCH} = time() + $run_digits * $unit_idx{$run_unit};

  if ($opt{ANALYZE}){
	  return; # no more initialization needed   
  }
  for(qw|API_TOKEN YBA_HOST CUST_UUID UNIV_UUID|){
     $opt{$_} or die "ERROR: Required parameter --$_ was not specified.\n";
  }
  my $curl_base_cmd = join " ", $opt{CURL}, 
             qq|-s --request GET --header 'Content-Type: application/json'|,
             qq|--header "X-AUTH-YW-API-TOKEN: $opt{API_TOKEN}"|,
             qq|--url $opt{YBA_HOST}/api/customers/$opt{CUST_UUID}/universes/$opt{UNIV_UUID}|;
  if ($opt{DEBUG}){
     print "DEBUG:CURL base CMD: $curl_base_cmd\n";
  }
  # Get universe name ..
  my $json_string = qx|$curl_base_cmd |;
  if ($?){
	print "ERROR: curl base command failed: $?\n";
	exit 1;
  }
  
  my %univ = $json_string=~m/"(\w[^"]+)":"([^"]+)/g;  # Grab all simple scalars
  %univ = %univ, $json_string=~m/"(\w[^"]+)":\[([^\]]+)\]/g; # Append all arrays (not decoded)
  $opt{DEBUG} and print "DEBUG: $_\t","=>",$univ{$_},"\n" for sort keys %univ;
  #my ($universe_name) =  $json_string =~m/,"name":"([^"]+)"/;
  if ($univ{universeName}){
	 print "UNIVERSE: ", $univ{universeName}," on ", $univ{providerType}, " ver ",$univ{ybSoftwareVersion},"\n";
  }else{
     $opt{DEBUG} and  print "DEBUG: Universe not found in curl_base:\n $json_string\n";
	 $json_string !~/{/ and print "ERROR?: $json_string\n";
  }
  
  $curl_cmd = $curl_base_cmd . "/live_queries"; # Henceforth - this is used. 
  # Run iteration ONCE, to verify it works...
  return unless $opt{DAEMON} ;
  
  print "Testing main loop before daemonizing...\n";
  if (Main_loop_Iteration()){
	  print "ERROR in main loop iteration. quitting...\n";
     exit 2;
  }
  # Close open file handles that may be leftover from main loop outputs
  for (keys %outinfo){ # Key is "type" - ycql or ysql
     next unless $outinfo{$_}{FH};
	 close $outinfo{$_}{FH};
	 delete $outinfo{$_}; # Force it to re-open 
  }
  print "End main loop test.\n";  
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
	    print unixtime_to_printable(time(),"YYYY-MM-DD HH:MM:SS")," querymonitor child process $grandchild started(from $grandpop_pid). exiting parent(s)\n";
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
	my ($unixtime,$format) = @_;
	my ($sec,$min,$hour,$mday,$mon,$year,$wday,$yday,$isdst) = localtime($unixtime);
	if (not defined $format  or  $format eq "YYYY-MM-DD HH:MM:SS"){
       return sprintf("%04d-%02d-%02d %02d:%02d:%02d", $year+1900, $mon+1, $mday, $hour, $min, $sec);
	}
	if ($format eq "YYYY-MM-DD HH:MM"){
       return sprintf("%04d-%02d-%02d %02d:%02d", $year+1900, $mon+1, $mday, $hour, $min);
	}	
	if ($format eq "YYYY-MM-DD"){
		  return sprintf("%04d-%02d-%02d", $year+1900, $mon+1, $mday);
	}
	die "ERROR: Unsupported format:'$format' ";
}
#------------------------------------------------------------------------------
sub Process_the_CSV_through_Sqlite{
	if ( -f $opt{ANALYZE} ){
		# File exists - fall through and process it
	}else{
	    die "ERROR: 'ANALYZE' file does not exist: No file '$opt{ANALYZE}'.";	
	}
	print "Analyzing $opt{ANALYZE} ...\n";
	
	my ($sqlite_version) = do {my $vv=qx|$opt{SQLITE} --version|;chomp $vv;$vv=~/([\d\.]+)/};
	$! and die "ERROR: Cannot run $opt{SQLITE} :$!";
	$sqlite_version or die "ERROR: could not get SQLITE version";
	# We do several extra steps to allow for OLD sqlite (< 3.8):
	# (a) It does not support instr(), so we use xx like yy
	# (b) the .import command does not allow --skip, so we pre-skip on input file
	
	# Use Forking and fifo to de-compress, remove first line, 
	# then feed via the fifo into the ".import" command in sqlite
	my $fifo = "/tmp/querymonitor_fifo_$$";
	print "Creating temporary fifo $fifo ...\n";
	qx|mkfifo $fifo|;
	$! and die "ERROR: Fifo failed: $!";
	
	#Fork to provide CSV stream to fifo
	my $extract_cmd = 'gunzip -c'; # expecting a gzipped file
	$opt{ANALYZE} !~/\.gz$/i and $extract_cmd = 'cat'; # It is NOT gzipped 
    my $pid = fork ();
    if ($pid < 0) {
      die "ERROR: fork failed: $!";
    } elsif ($pid) {
	  # This is the parent --fall through 
    }else{
		# This is the CHILD 
	    #close std fds inherited from parent
		close STDIN;
		close STDOUT;
		close STDERR;		
		qx{$extract_cmd $opt{ANALYZE} | sed 1d > $fifo};
		exit 0; # Exit child process and close FIFO
	}
	
	if (! $opt{DB}){
	   # DB name was not specified .. generate it from ANALYZE file name
	   $opt{DB} = $opt{ANALYZE};
	   $opt{DB} =~s/\.gz$//i;  # drop .gz
	   $opt{DB} =~s/\.csv$//i; # drop .csv
	   $opt{DB} .= ".sqlite";  # append .sqlite 
	}
	print "Creating sqlite database $opt{DB} ...\n";
	
	open my $sqlfh ,  "|-" , $opt{SQLITE} , $opt{DB} or die "ERROR: Could not run SQLITE";
	print $sqlfh <<"__SQL__";
.version
.header off
CREATE TABLE q (ts integer,clientHost,clientPort,elapsedMillis integer,id,keyspace,nodeName,privateIp,query,type);
CREATE VIEW summary as select datetime((ts/600)*600,'unixepoch','-4 Hours') as EDT,
    sum(case when query LIKE '% system.%'  then 1 else 0 end) as systemq,
        sum(case when query LIKE '% system.%' then 0 else 1 end) as cqlcount,
        sum(case when query LIKE '% system.%' and elapsedmillis > 120 then 1 else 0 end) as sys_gt120,
        sum(case when query NOT LIKE '% system.%' and elapsedmillis > 120 then 1 else 0 end) as cql_gt120,
        sum(case when query LIKE '% system.%' and nodeName
            in ('yb-prod-CAS-PROD-Primary-n2','yb-prod-CAS-PROD-Primary-n3','yb-prod-CAS-PROD-Primary-n1','yb-prod-CAS-PROD-Primary-n5',
			    'yb-prod-CAS-PROD-Secondary-n1','yb-prod-CAS-PROD-Secondary-n5','yb-prod-CAS-PROD-Secondary-n6','yb-prod-CAS-PROD-Secondary-n7',
				'yb-prod-CAS-PROD-Secondary-n9','yb-prod-CAS-PROD-Secondary-n10','yb-prod-CAS-PROD-Secondary-n11','yb-prod-CAS-PROD-Secondary-n12')
          then 1 else 0 end) as sys_dc3,
        sum(case when query NOT LIKE '% system.%' and nodeName
            in ('yb-prod-CAS-PROD-Primary-n2','yb-prod-CAS-PROD-Primary-n3','yb-prod-CAS-PROD-Primary-n1','yb-prod-CAS-PROD-Primary-n5')
          then 1 else 0 end) as cql_dc3 ,
         round(sum(case when query NOT LIKE '% system.%' and elapsedmillis > 120 then 1 else 0 end) *100.0
               / sum(case when query NOT LIKE '% system.%' then 1 else 0 end)
                   ,2) as breach_pct
FROM q group by EDT;

CREATE VIEW query_perf as  select query, count(*) as nbr_querys, round(avg(elapsedmillis),1) as avg_milli ,
      sum (CASE when elapsedmillis > 120 then 1 else 0 END)*100 / count(*) as pct_gt120,
          sum ( CASE WHEN nodeName
            in ('yb-prod-CAS-PROD-Primary-n2','yb-prod-CAS-PROD-Primary-n3','yb-prod-CAS-PROD-Primary-n1','yb-prod-CAS-PROD-Primary-n5',
			    'yb-prod-CAS-PROD-Secondary-n1','yb-prod-CAS-PROD-Secondary-n5','yb-prod-CAS-PROD-Secondary-n6','yb-prod-CAS-PROD-Secondary-n7',
				'yb-prod-CAS-PROD-Secondary-n9','yb-prod-CAS-PROD-Secondary-n10','yb-prod-CAS-PROD-Secondary-n11','yb-prod-CAS-PROD-Secondary-n12')
                THEN 1 ELSE 0 END) as DC3_querys
          FROM q
          GROUP BY query
          HAVING nbr_querys > 50 and avg_milli >30  ORDER by avg_milli  desc;
.mode csv
.import '$fifo' q
.mode column
SELECT 'Imported ' || count(*) ||' rows from $opt{ANALYZE}.' as Imported_count from q;
SELECT ''; -- blank line
SELECT '====== Summary Report ====';
.header on
SELECT * from summary;
.header off
SELECT '';
SELECT '======= Slow Queries =======';
.header on
select * from query_perf;
.q
__SQL__

   close $sqlfh;
   unlink $fifo;
   wait; # For kid 
}
#==============================================================================
#==============================================================================