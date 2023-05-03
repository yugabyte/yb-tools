#!/usr/bin/perl

our $VERSION = "0.13";
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
}

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
	OUTPUT                        => "queries." . unixtime_to_printable(time(),"YYYY-MM-DD") . ".mime.gz",
    # Misc
    DEBUG                         => 0,
    HELP                          => 0,
    DAEMON                        => 1,
    LOCKFILE                      => "/var/lock/querymonitor.lock", # UNIV_UUID will be appended
    LOCK_FH                      => undef,
	MAX_QUERY_LEN                => 2048,
	MAX_ERRORS                   => 10,
	SANITIZE					 => 0,   # Remove PII 
	ANALYZE						 => undef,     # Input File-name ( "..csv.gz" ) to process through sqlite
	DB                           => undef,     # output SQLITE database file name
	SQLITE                       => "sqlite3", # path to Sqlite binary
	UNIVERSE                     => undef,     # Universe detail info (Populated in initialize)
	HTTPCONNECT                  => "tiny",    # How to connect to the YBA : "curl", or "tiny" (HTTP::Tiny)
);

my $quit_daemon = 0;
my $loop_count  = 0;
my $error_counter=0;
my $YBA_API;  # Populated in `Initialize` to a Web::Interface object
my $output;   # Populated in `Initialize to a MIME::Write::Simple object
my $curl_cmd; # Populated in `Initialize`

Initialize();

if ($opt{ANALYZE}){
   Analysis::Mode::->Process_the_CSV_through_Sqlite();
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
$output->Close();

exit 0;
#------------------------------------------------------------------------------
#==============================================================================
#------------------------------------------------------------------------------
sub Main_loop_Iteration{

    my $query_type = "Unknown";

    if ($opt{DEBUG}){
        print "DEBUG: Start main loop iteration $loop_count\n";
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
             #print join(",",$ts, $type,$q->{nodeName},$q->{id},$subquery),"\n";
			 $output->WriteQuery($ts, $type, $q, $sanitized_query);
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


#------------------------------------------------------------------------------
sub Initialize{
  chomp ($opt{HOSTNAME} = qx|hostname|);
  print $opt{STARTTIME}," Starting $0 version $VERSION  PID $$ on $opt{HOSTNAME}\n";

  my @program_options = qw[ DEBUG! HELP! DAEMON! SANITIZE!
                       API_TOKEN=s YBA_HOST=s CUST_UUID=s UNIV_UUID=s
                       INTERVAL_SEC=i RUN_FOR|RUNFOR=s CURL=s SQLITE=s
                       FLAGFILE=s OUTPUT=s DB=s
					   MAX_QUERY_LEN|MAXQL|MQL=i ANALYZE|PROCESS=s
					   HTTPCONNECT=s];
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
  if (@ARGV){
    die "ERROR: Unknown argument (flag expected) : @ARGV";	  
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

  $YBA_API = Web::Interface::->new();

  # Get universe name ..
  $opt{UNIVERSE} = $YBA_API->Get("");

  $opt{DEBUG} and print "DEBUG: $_\t","=>",$opt{UNIVERSE}{$_},"\n" for sort keys %{$opt{UNIVERSE}};
  #my ($universe_name) =  $json_string =~m/,"name":"([^"]+)"/;
  if ($opt{UNIVERSE}{name}){
	 print "UNIVERSE: ", $opt{UNIVERSE}{name}," on ", $opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{providerType}, " ver ",$opt{UNIVERSE}{universeDetails}{clusters}[0]{userIntent}{ybSoftwareVersion},"\n";
  }else{
     $opt{DEBUG} and  print "DEBUG: Universe info not found \n";
  }
  $opt{NODEHASH} = {map{$_->{nodeName} => {%{$_->{cloudInfo}}, uuid=>$_->{nodeUuid} ,state=>$_->{state},isTserver=>$_->{isTserver}}} 
						@{ $opt{UNIVERSE}->{universeDetails}{nodeDetailsSet} } };

  $output = MIME::Write::Simple::->new(); # No I/O so far 
  # Run iteration ONCE, to verify it works...
  return unless $opt{DAEMON} ;
  
  $opt{LOCKFILE} .= ".$opt{UNIV_UUID}"; # one lock per universe 
  print "Testing main loop before daemonizing...\n";
  if (Main_loop_Iteration()){
	  print "ERROR in main loop iteration. quitting...\n";
     exit 2;
  }
  # Close open file handles that may be leftover from main loop outputs
  $output->Close();
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

#==============================================================================
BEGIN{
package Analysis::Mode; 

sub new{ # Unused
    my ($class) = @_;
    return bless {}, $class;	
}

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
	my $populate_zone_map=""; # Need a better way to do this...
	if ($opt{ANALYZE} =~/SECONDARY/i){
		$populate_zone_map = <<"__SEC_ZONE__";
		update zone_map set zone='DC3';
		update zone_map set zone='DC1' where node like '%n2';
		update zone_map set zone='DC1' where node like '%n3';
		update zone_map set zone='DC1' where node like '%n4';
		update zone_map set zone='DC1' where node like '%n9';
__SEC_ZONE__
    }elsif ( $opt{ANALYZE} =~/PRIMARY/i ){
		$populate_zone_map = <<"__PRI_ZONE__";
		update zone_map set zone='DC1';
		update zone_map set zone='DC3' where node like '%n1';
		update zone_map set zone='DC3' where node like '%n2';
		update zone_map set zone='DC3' where node like '%n3';
		update zone_map set zone='DC3' where node like '%n5';
__PRI_ZONE__
	}
    
	
	open my $sqlfh ,  "|-" , $opt{SQLITE} , $opt{DB} or die "ERROR: Could not run SQLITE";
	print $sqlfh <<"__SQL__";
.version
.header off
CREATE TABLE IF NOT EXISTS run_info(key text, value text);
INSERT INTO run_info VALUES ('data file','$opt{ANALYZE}')
      ,('HOSTNAME','$opt{HOSTNAME}')
	  ,('import date',datetime('now','localtime'));
CREATE TABLE zone_map (node,zone);	  
CREATE TABLE q (ts integer,clientHost,clientPort,elapsedMillis integer,id,keyspace,nodeName,privateIp,query,type);
CREATE VIEW summary as select datetime((ts/600)*600,'unixepoch','-4 Hours') as EDT,
    sum(case when instr(query,' system.')> 0 then 1 else 0 end) as systemq,
        sum(case when instr(query,' system.')=0 then 1 else 0 end) as cqlcount,
        sum(case when instr(query,' system.')>0 and elapsedmillis > 120 then 1 else 0 end) as sys_gt120,
        sum(case when instr(query,' system.')=0 and elapsedmillis > 120 then 1 else 0 end) as cql_gt120,
        sum(case when instr(query,' system.')>0 and zone='DC1' then 1 else 0 end) as sys_dc1,
        sum(case when instr(query,' system.')=0 and zone='DC1' then 1 else 0 END) as cql_dc1,
        sum(case when instr(query,' system.')>0 and zone='DC3' then 1 else 0 end) as sys_dc3,
        sum(case when instr(query,' system.')=0 and zone='DC3' then 1 else 0 END) as cql_dc3,		
         round(sum(case when instr(query,' system.')=0 and elapsedmillis > 120 then 1 else 0 end) *100.0
               / sum(case when instr(query,' system.')=0 then 1 else 0 end)
                   ,2) as breach_pct
FROM q,zone_map
   where q.nodename=zone_map.node 
 group by EDT;

CREATE VIEW slow_queries  as  select query, count(*) as nbr_querys, round(avg(elapsedmillis),1) as avg_milli ,
      sum (CASE when elapsedmillis > 120 then 1 else 0 END)*100 / count(*) as pct_gt120,
          sum ( CASE WHEN zone = 'DC1' THEN 1 ELSE 0 END) as dc1_queries,
		  sum ( CASE WHEN zone = 'DC3' THEN 1 ELSE 0 END) as dc3_queries
          FROM q, zone_map
		   where q.nodename=zone_map.node 
          GROUP BY query
          HAVING nbr_querys > 50 and avg_milli >30  ORDER by avg_milli  desc;
		  
CREATE VIEW NODE_Report AS  select nodename, round(avg(elapsedmillis),1) as avg_ms, 
       count(*), sum(case when instr(query,' system.') > 0 then 1 else 0 end) as sys_count,  
	   sum(case when instr(query,' system.')= 0 then 1 else 0 end) as cql_count,
	   sum(case when instr(query,' system.')= 0 and elapsedmillis > 120 then 1 else 0 end) as cql_gt_120,
	   sum(case when instr(query,' system.')> 0 and elapsedmillis > 120 then 1 else 0 end) as sys_gt_120,
	   zone
  from q,zone_map 
  where nodename=node  
  group by  nodename 
  order by nodename;
  
.mode csv
.import '$fifo' q
.mode column
SELECT 'Imported ' || count(*) ||' rows from $opt{ANALYZE}.' as Imported_count from q;

insert into zone_map select distinct nodename,'UNKNOWN' from q;
$populate_zone_map;
SELECT ''; -- blank line
SELECT '====== Summary Report ====';
.header on
SELECT * from summary;
.header off
SELECT '';
SELECT '======= Slow Queries =======';
.header on
select * from slow_queries;
.q
__SQL__

   close $sqlfh;
   unlink $fifo;
   wait; # For kid 
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
	my $self =bless {map {$_ => $opt{$_}} qw|HTTPCONNECT UNIV_UUID API_TOKEN YBA_HOST CUST_UUID| }, $class;
	$self->{BASE_URL} = "$opt{YBA_HOST}/api/customers/$opt{CUST_UUID}/universes/$opt{UNIV_UUID}";
	if ($self->{HTTPCONNECT} eq "curl"){
		  $self->{curl_base_cmd} = join " ", $opt{CURL}, 
					 qq|-s --request GET --header 'Content-Type: application/json'|,
					 qq|--header "X-AUTH-YW-API-TOKEN: $opt{API_TOKEN}"|,
					 qq|--url $self->{BASE_URL}|;
		  if ($opt{DEBUG}){
			 print "DEBUG:CURL base CMD: $self->{curl_base_cmd}\n";
		  }
		  return $self;
    }
    
	$self->{HT} = HTTP::Tiny->new( default_headers => {
                         'X-AUTH-YW-API-TOKEN' => $opt{API_TOKEN},
						 'Content-Type'      => 'application/json',
	                  });

    return $self;
}

sub Get{
	my ($self, $endpoint) = @_;
	$self->{json_string}= "";
	$self->{response} = undef;
	if ($self->{HTTPCONNECT} eq "curl"){
		$self->{json_string} = qx|$self->{curl_base_cmd}$endpoint|;
		if ($?){
		   print "ERROR: curl get '$endpoint' failed: $?\n";
		   exit 1;
		}
    }else{ # HTTP::Tiny
	   $self->{raw_response} = $self->{HT}->get( 'http://' . $self->{BASE_URL} . $endpoint );
	   if (not $self->{raw_response}->{success}){
		  print "ERROR: Get '$endpoint' failed with status=$self->{raw_response}->{status}: $self->{raw_response}->{reason}\n";
		  exit 1;
	   }
	   $self->{json_string} = $self->{raw_response}{content};
	}
	$self->{json_string}=~s/:/=>/g;
	$self->{json_string}=~s/(\W)true(\W)/${1}1$2/g;
	$self->{json_string}=~s/(\W)false(\W)/${1}0$2/g;
	if (length($endpoint) == 0){
	    # Special case for UNIVERSE .. clean unparsable...
        $self->{json_string}=~s/"sampleAppCommandTxt"=>.+",//;
	}
    $@="";
    $self->{response} = eval $self->{json_string};
	$@ and die "EVAL ERROR getting $endpoint:$@";
    #my %univ = $json_string=~m/"(\w[^"]+)":"([^"]+)/g;  # Grab all simple scalars
    #%univ = %univ, $json_string=~m/"(\w[^"]+)":\[([^\]]+)\]/g; # Append all arrays (not decoded)
	return $self->{response};
}

} # End of  package Web::Interface;  
#==============================================================================
BEGIN{
package MIME::Write::Simple;

sub new{
	my ($class,%att) = @_;
	
	$att{boundary} ||= "--" . time() . $$;
	$att{MIME_VERSION_SENT} = 0;
	return bless {%att},$class;
}

sub header{
	my ($self,$content_type,$header_msg) = @_;
	$header_msg ||= "";
	chomp $header_msg;
	$header_msg and $header_msg .= "\n";
	my $mime_ver = $self->{MIME_VERSION_SENT} ? "" 
                   : "MIME-Version: 1.0\n";
    $self->{MIME_VERSION_SENT} = 1;
	print { $self->{FH} }   $mime_ver
	        . "Content-Type: $content_type;\n"
          . qq|  boundary="$self->{boundary}"\n\n|
		  . $header_msg ;
}

sub boundary{ # getter/setter
   my ($self,$b, $final) = @_;
   $b and $self->{boundary} = $b;
   print { $self->{FH} }  $self->{boundary} . ($final ? "--\n" : "\n");
}

sub Open_and_Initialize{
	my ($self) = @_;
	$opt{DEBUG} and print "DEBUG: Opening output file=" , $opt{OUTPUT},"\n";
	my $output_already_exists = -f $opt{OUTPUT};
	open $self->{FH} , "|-", "gzip -c >> " . $opt{OUTPUT}
	  or die "ERROR: Cannot fork output zip:$!";
	if (! $output_already_exists){
	   # Heed MIME headers etc...
	   $self->header("multipart/mixed",
	      join ("\n",
			  "Querymonitor_version: $VERSION",
			  "UNIVERSE: $opt{UNIVERSE}{name}",
			  "UNIV_UUID: $opt{UNIV_UUID}",
			  "STARTTIME: $opt{STARTTIME}",
			  "Run_host: $opt{HOSTNAME}"
		  ));
	    boundary();  
        # Insert NODE/ZONE info
 
	}
}

sub Initialize_query_type{
	my ($self,$type,$q) = @_;
    if ($self->{IN_CSV_SECTION}){
	   $self->boundary(); # Close the CSV section
	}
	$self->header("text/csvheader",
	      join("\n",
		     "TYPE: $type",
			 "FIELDS: " . join(",","ts",sort keys %$q)
			 ));
	$self->boundary();
	$self->{TYPE_INITIALIZED}{$type} = 1;
	$self->header("text/csv");
	$self->{IN_CSV_SECTION} = 1;
}

sub WriteQuery{
  my ($self,$ts, $type, $q, $sanitized_query) = @_;
  
  if (! $self->{FH} ){
	  # Need to initialize output
      $self->Open_and_Initialize();
  }
  if ( ! $self->{TYPE_INITIALIZED}{$type} ){
	  $self->Initialize_query_type($type, $q);  
  }
  $sanitized_query =~tr/"/~/; # Zap internal double quotes - which will mess CSV
  if (length($sanitized_query) > $opt{MAX_QUERY_LEN}){
	   $sanitized_query = substr($sanitized_query,0,$opt{MAX_QUERY_LEN}/2 -2) 
	                     . ".." . substr($sanitized_query,-($opt{MAX_QUERY_LEN}/2));
  }
  $q->{query} = qq|"$sanitized_query"|;
  print { $self->{FH} } join(",", $type, $ts, map( {$q->{$_}} sort keys %$q)),"\n";
}

sub Close{
	my ($self) = @_;
    return unless $self->{FH};
	$self->boundary(undef,"FINAL");
	$self->{IN_CSV_SECTION} = 0;
    close $self->{FH};
    $self->{FH} = undef;	
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


=head1 SYNOPSIS

This is a really basic MIME multipart parser, 
and the only reason for its existence is that
I could not find an existing parser that would
give me the parts directly (not on fs) and also
give me the order.

	my $mmps = MIME::Multipart::ParseSimple->new();
	my $listref = $mmps->parse($my_file_handle);
	print $listref->[0]->{"Preamble"};
	print $listref->[0]->{"Content-Type.params"}->{"boundary"};
	foreach (@$listref){
		print $_->{"Body"} 
		  if $_->{"Content-Type"} eq 'text/plain';
	}
=cut

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
  my ($o,$fh) = @_;
  $o->{fh} = $fh;

  my $mp1 = <$fh>;
  my $mp1e = 'MIME Version: 1.0';
  die "Multipart header line 1 must begin ``$mp1e'' " unless $mp1 =~ /^$mp1e/;
 
  my $general_header = $o->parseHeader();
  croak "no boundary defined" unless $general_header->{"Content-Type.params"}->{"boundary"};
  $o->{boundary} = $general_header->{"Content-Type.params"}->{"boundary"};
  
  $general_header->{Preamble} = $o->parseBody();

  my @parts = ($general_header);

  while(! (eof($fh) || $o->{eof})){
    my $header = $o->parseHeader();
    $header->{Body} = $o->parseBody();
    push @parts, $header;
  }

  $general_header->{Epilog} = $o->parseBody();

  return \@parts;

}

sub parseBody {
  my ($o) = @_;
  my $fh = $o->{fh};
  my $body = '';
  my $boundary = $o->{boundary};
  while(<$fh>){
    $o->{eof} = 1 if /^--$boundary--/;
    last if /^--$boundary/;
    $body .= $_;
  }
  return $body;
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
      $k =~ s/(?:^\s+|\s+$)//g;
      $v =~ s/(?:^\s+|\s+$)//g;
      $header{$k} = $v;
      $p = $k.'.params';
      $header{$p} = {};
    }
    foreach my $part(@parts){
      my ($l,$w) = split(/=/, $part, 2);
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