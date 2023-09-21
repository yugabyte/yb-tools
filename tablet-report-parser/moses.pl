#!/usr/bin/perl

our $VERSION = "0.03";
my $HELP_TEXT = << "__HELPTEXT__";
    moses.pl  Version $VERSION
    ===============
 Get and analyze info on all tablets in the system.
 collect gzipped JSON file for offline analysis

__HELPTEXT__
use strict;
use warnings;
use Getopt::Long;
use Time::Piece;
use Time::Seconds;
use HTTP::Tiny;

{package HTML::TagParser;
 package Tablet;
 package Web::Interface;
 package JSON::Tiny;
 package OutputMechanism;
 package DatabaseClass;
}; # Pre-declare 

my %opt = (
   STARTTIME            => scalar (Time::Piece::->localtime()),
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
   UNIVERSE             => undef,
);

#---- Start ---
my ($YBA_API);
Initialize();

Get_and_Parse_tablets_from_tserver();

exit 0;
#----------------------------------------------------------------------------------------------
sub Get_and_Parse_tablets_from_tserver{

  for my $n (@{ $opt{NODES} }){
      next unless $n->{isTserver};
      if ( $n->{state} ne  'Live'){
         print "-- Node $n->{nodeName} $n->{nodeUuid} is $n->{state} .. skipping\n";
         next;
      }

      print "-- Processing tablets on $n->{nodeName} $n->{nodeUuid} (Idx $n->{nodeIdx})...\n";
      my $html_raw = $YBA_API->Get("/proxy/$n->{private_ip}:$n->{tserverHttpPort}/tablets?raw","BASE_URL_UNIVERSE",1); # RAW
      my $row   = 0;
      my $html  = HTML::TagParser->new( $html_raw );
      my $table = $html->getElementsByTagName("table");
      #  <tr><th>Namespace</th><th>Table name</th><th>Table UUID</th><th>Tablet ID</th><th>Partition</th><th>State</th>
      #      <th>Hidden</th><th>Num SST Files</th><th>On-disk size</th><th>RaftConfig</th><th>Last status</th></tr>
      my $tr = my $header= $table->firstChild();  # Header row
      Tablet::->SetFieldNames( map {$_->innerText() } @{$header->childNodes()} );
      
      
      my (@tabs, %leaders);
      while ( $tr = $tr->nextSibling() ){
      	my $t =  Tablet::->new($tr);
      	#$t->Print();
      	#print "\n";
      	#last if $row++ > 5;
      	$leaders{$t->{RAFTCONFIG}{LEADER}} ++;
      	push @tabs, $t;
      }
      
      print "Found ",scalar(@tabs)," tablets\n";
      print "$leaders{$_}\t tablets on leader $_\n" for sort keys %leaders;
  }

}
#----------------------------------------------------------------------------------------------

#----------------------------------------------------------------------------------------------
sub Initialize{
    print "-- ",$opt{STARTTIME}," Starting $0 version $VERSION  PID $$ on $opt{LOCALHOST}\n";

    GetOptions (\%opt, qw[DEBUG! HELP! VERSION!
                        API_TOKEN=s YBA_HOST=s UNIVERSE=s
                        HTTPCONNECT=s CURL=s]
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
      warn "\n  --RUN Options:\n",
          map({"      $_ (@{$DatabaseClass::view_names{$_}{ALT}}): $DatabaseClass::view_names{$_}{TEXT}\n "} 
              grep {$DatabaseClass::view_names{$_}} sort keys %DatabaseClass::view_names), "\n";
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
     print "--UNIVERSE: ", $opt{UNIV_DETAILS}{name}," on ", $opt{UNIV_DETAILS}{universeDetails}{clusters}[0]{userIntent}{providerType},
           " ver ",$opt{UNIV_DETAILS}{universeDetails}{clusters}[0]{userIntent}{ybSoftwareVersion},"\n";
  }else{
     die "ERROR: Universe info not found \n";
  }
  $opt{NODES} = Extract_nodes_From_Universe($opt{UNIV_DETAILS});
  # Find Master/Leader 
  $opt{MASTER_LEADER}      = $YBA_API->Get("/leader")->{privateIP};
  $opt{DEBUG} and print "--DEBUG:Master/Leader JSON:",$YBA_API->{json_string},". IP is ",$opt{MASTER_LEADER},".\n";
  my ($ml_node) = grep {$_->{private_ip} eq $opt{MASTER_LEADER}} @{ $opt{NODES} } or die "ERROR : No Master/Leader NODE found for $opt{MASTER_LEADER}";
  my $master_http_port = $opt{UNIV_DETAILS}{universeDetails}{communicationPorts}{masterHttpPort} or die "ERROR: Master HTTP port not found in univ JSON";
  # Get dump_entities JSON from MASTER_LEADER
  $opt{DEBUG} and print "--DEBUG:Getting Dump Entities...\n";  
  $YBA_API->Get("/proxy/$ml_node->{private_ip}:$master_http_port/dump-entities","BASE_URL_UNIVERSE");
  # Analyze & save $YBA_API->{json_string} 
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
###############################################################################
############### C L A S S E S                             #####################
###############################################################################
BEGIN{
package OutputMechanism;
use warnings;
use strict;
sub new{
  my ($class, %att) = @_;
	die "ERROR:No output filename/DB specified" unless $att{FILENAME} or $att{TO_STDOUT} or $att{DBFILE};
	my $self = {recordcount=>0, %att};
	if ($att{GZIP}){
	    open $self->{OUTPUT_FH} , "|/bin/gzip > $att{FILENAME}" or die "ERROR:Could not open output gz $att{FILENAME} :$!";
  }elsif ( $self->{TO_STDOUT} ){
      # STDOUT is already open . Nothing needed
  }elsif ( $att{DBFILE} ){ # Request to actually create the sqlite db to this file
      open $self->{OUTPUT_FH} , "|$att{SQLITE} $att{DBFILE}" or die "ERROR:Could not open output SQLITE $att{DBFILE} :$!";
	}else{
		# Non-gzip output requested - just write to simple file handle
		open $self->{OUTPUT_FH} , ">", $att{FILENAME} or die "ERROR:Could not open output file $att{FILENAME} :$!";
	}
    return bless $self, $class;
}
sub send{
   my  ($self,@msg) = @_;
   if ( $self->{TO_STDOUT} ){
       print  "$_\n" for @msg; 
   }else{
       print {$self->{OUTPUT_FH}}  "$_\n" for @msg;
   }
   $self->{recordcount}+=$#msg + 1;
}
sub close{
   my  ($self) = @_;
   return 0 if $self->{TO_STDOUT}; # STDOUT will auto-close
   close $self->{OUTPUT_FH};
   chmod 0644, grep {defined}  $self->{FILENAME} , $self->{DBFILE};
}
1;
}
#=====================================================================================
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
    NODE => { FIELDS => [qw|ipaddr macaddr nodes hfscreatetime systemid timestamp HOSTNAME ISSOURCEGRID|],
      CREATE=> <<"     __GRID__",
       CREATE TABLE  IF NOT EXISTS node(
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
       CREATE UNIQUE INDEX IF NOT EXISTS by_gridid ON GRID (id);
       CREATE UNIQUE INDEX IF NOT EXISTS by_sysid  ON GRID (systemid);
     __GRID__
     DROPPABLE => 1,
    },TAABLE => {
       CREATE=><<"     __DOMAIN__",
       CREATE TABLE  IF NOT EXISTS domain(
       id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
       gridid INTEGER NOT NULL REFERENCES GRID(id) ON DELETE CASCADE,
       name     TEXT NOT NULL,
	   cid      TEXT,
       parentid INTEGER,
       CONTACTINFO TEXT
       );
       CREATE UNIQUE INDEX IF NOT EXISTS by_domainid ON DOMAIN (id);  
       CREATE UNIQUE INDEX IF NOT EXISTS by_gridid ON DOMAIN (gridid,id); 
       CREATE UNIQUE INDEX IF NOT EXISTS by_domain_name ON DOMAIN (name,gridid);  
     __DOMAIN__
     DROPPABLE => 1,
    }, 
    
);


sub new {
    my ($class,%att) = @_;
    $opt{DEBUG} and print "--- Creating NEW DatabaseClass object\n";
    $class = ref $class if ref $class;
    $att{OUTPUTOBJ} or die "ERROR: OUTPUTOBJ attribute not specified";

    my $self = bless \%att , $class;
    $opt{SQLFILENAME} ||="";
    $self->putsql("-- Generating $opt{SQLFILENAME} by $0 $VERSION on $opt{LOCALHOST} by " . ($ENV{USER}||$ENV{USERNAME}) . " on " . scalar(localtime(time)));
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
    $self->putsql(DB_Views());
    my $t = time();
    $self->putlog($_,7,$t) for (
         "--STARTUP --",
         "PROG_VERSION $VERSION",
         "SCHEMA_VERSION $SCHEMA_VERSION",
         "LOCALHOST $opt{LOCALHOST}",
         "USER " .($ENV{USER}||$ENV{USERNAME}),
         "DATE " .  $opt{STARTTIME}->ymd() . " ". $opt{STARTTIME}->hms()
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
  $self->{OUTPUTOBJ}->send ($txt) ; # SEND will append the "\n"
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
1;  
} # End of package DatabaseClass
###############################################################################
##########################################################################################################
BEGIN{
package Tablet;
my @fields;

sub new{
  my ($class,$tr) = @_;
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
   my $fld_idx = 0;
   bless my $self={}, $class;
   my $td = $tr->firstChild();
   while ($td){
      my $ul = $td->firstChild();
      if ($ul and $ul->tagName() eq "ul"){
         my $outer_field = $fields[$fld_idx++];
         my $li = $ul->firstChild();
         while($li ){ #and $li->tagName eq "li"){
            if ( my $txt = $li->innerText()){
               my ($subfield,$val) = $txt =~/\s*([^:]+):\s*(.+)/;
               if ($subfield eq "FOLLOWER"){
                  push @{$self->{$outer_field}{$subfield} }, $val;
               }else{
                  $self->{$outer_field}{$subfield} = $val;
               }
            }
            $li = $li->nextSibling();
         }
      }else{
         $self->{$fields[$fld_idx++]} = $td->innerText();
      }
      $td = $td->nextSibling();
   }
  return $self;
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
sub SetFieldNames{
  my ($class,@names_raw) = @_;
  @fields = map {tr/ //d; uc $_} @names_raw;
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
    my ($self, $endpoint, $base, $raw) = @_;
    $self->{json_string}= "";
    my $url = $base ? $self->{$base} : $self->{BASE_URL_API_CUSTOMER};
    if ($self->{HTTPCONNECT} eq "curl"){
        $self->{json_string} = qx|$self->{curl_base_cmd} --url $url$endpoint|;
        if ($?){
           print "ERROR: curl get '$endpoint' failed: $?\n";
           return {error=>$?};
        }
    }else{ # HTTP::Tiny
       $self->{raw_response} = $self->{HT}->get($url . $endpoint);
       if (not $self->{raw_response}->{success}){
          print "ERROR: Get '$endpoint' failed with status=$self->{raw_response}->{status}: $self->{raw_response}->{reason}\n\tURL:$url$endpoint\n";
          $self->{raw_response}->{status} == 599 and print "\t(599)Content:$self->{raw_response}{content};\n";
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
BEGIN{
package HTML::TagParser;
use 5.008_001;
use strict;
use Symbol ();
use Carp ();
use Encode ();
 
our $VERSION = "0.20";
 
my $SEC_OF_DAY = 60 * 60 * 24;
 
#  [000]        '/' if closing tag.
#  [001]        tagName
#  [002]        attributes string (with trailing /, if self-closing tag).
#  [003]        content until next (nested) tag.
#  [004]        attributes hash cache.
#  [005]        innerText combined strings cache.
#  [006]        index of matching closing tag (or opening tag, if [000]=='/')
#  [007]        index of parent (aka container) tag.
#
sub new {
    my $package = shift;
    my $src     = shift;
    my $self    = {};
    bless $self, $package;
    return $self unless defined $src;
 
    if ( $src =~ m#^https?://\w# ) {
        $self->fetch( $src, @_ );
    }
    elsif ( $src !~ m#[<>|]# && -f $src ) {
        $self->open($src);
    }
    elsif ( $src =~ /<.*>/ ) {
        $self->parse($src);
    }
 
    $self;
}
 
sub fetch {
    my $self = shift;
    my $url  = shift;
    if ( !defined $URI::Fetch::VERSION ) {
        local $@;
        eval { require URI::Fetch; };
        Carp::croak "URI::Fetch is required: $url" if $@;
    }
    my $res = URI::Fetch->fetch( $url, @_ );
    Carp::croak "URI::Fetch failed: $url" unless ref $res;
    return if $res->is_error();
    $self->{modified} = $res->last_modified();
    my $text = $res->content();
    $self->parse( \$text );
}
 
sub open {
    my $self = shift;
    my $file = shift;
    my $text = HTML::TagParser::Util::read_text_file($file);
    return unless defined $text;
    my $epoch = ( time() - ( -M $file ) * $SEC_OF_DAY );
    $epoch -= $epoch % 60;
    $self->{modified} = $epoch;
    $self->parse( \$text );
}
 
sub parse {
    my $self   = shift;
    my $text   = shift;
    my $txtref = ref $text ? $text : \$text;
 
    my $charset = HTML::TagParser::Util::find_meta_charset($txtref);
    $self->{charset} ||= $charset;
    if ($charset && Encode::find_encoding($charset)) {
        HTML::TagParser::Util::encode_from_to( $txtref, $charset, "utf-8" );
    }
    my $flat = HTML::TagParser::Util::html_to_flat($txtref);
    Carp::croak "Null HTML document." unless scalar @$flat;
    $self->{flat} = $flat;
    scalar @$flat;
}
 
sub getElementsByTagName {
    my $self    = shift;
    my $tagname = lc(shift);
 
    my $flat = $self->{flat};
    my $out = [];
    for( my $i = 0 ; $i <= $#$flat ; $i++ ) {
        next if ( $flat->[$i]->[001] ne $tagname );
        next if $flat->[$i]->[000];                 # close
        my $elem = HTML::TagParser::Element->new( $flat, $i );
        return $elem unless wantarray;
        push( @$out, $elem );
    }
    return unless wantarray;
    @$out;
}
 
sub getElementsByAttribute {
    my $self = shift;
    my $key  = lc(shift);
    my $val  = shift;
 
    my $flat = $self->{flat};
    my $out  = [];
    for ( my $i = 0 ; $i <= $#$flat ; $i++ ) {
        next if $flat->[$i]->[000];    # close
        my $elem = HTML::TagParser::Element->new( $flat, $i );
        my $attr = $elem->attributes();
        next unless exists $attr->{$key};
        next if ( $attr->{$key} ne $val );
        return $elem unless wantarray;
        push( @$out, $elem );
    }
    return unless wantarray;
    @$out;
}
 
sub getElementsByClassName {
    my $self  = shift;
    my $class = shift;
    return $self->getElementsByAttribute( "class", $class );
}
 
sub getElementsByName {
    my $self = shift;
    my $name = shift;
    return $self->getElementsByAttribute( "name", $name );
}
 
sub getElementById {
    my $self = shift;
    my $id   = shift;
    return scalar $self->getElementsByAttribute( "id", $id );
}
 
sub modified {
    $_[0]->{modified};
}
 
# ----------------------------------------------------------------
 
package HTML::TagParser::Element;
use strict;
 
sub new {
    my $package = shift;
    my $self    = [@_];
    bless $self, $package;
    $self;
}
 
sub tagName {
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    return $flat->[$cur]->[001];
}
 
sub id {
    my $self = shift;
    $self->getAttribute("id");
}
 
sub getAttribute {
    my $self = shift;
    my $name = lc(shift);
    my $attr = $self->attributes();
    return unless exists $attr->{$name};
    $attr->{$name};
}
 
sub innerText {
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $elem = $flat->[$cur];
    return $elem->[005] if defined $elem->[005];    # cache
    return if $elem->[000];                         # </xxx>
    return if ( defined $elem->[002] && $elem->[002] =~ m#/$# ); # <xxx/>
 
    my $tagname = $elem->[001];
    my $closing = HTML::TagParser::Util::find_closing($flat, $cur);
    my $list    = [];
    for ( ; $cur < $closing ; $cur++ ) {
        push( @$list, $flat->[$cur]->[003] );
    }
    my $text = join( "", grep { $_ ne "" } @$list );
    $text =~ s/^\s+|\s+$//sg;
#   $text = "" if ( $cur == $#$flat );              # end of source
    $elem->[005] = HTML::TagParser::Util::xml_unescape( $text );
}
 
sub subTree
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $elem = $flat->[$cur];
    return if $elem->[000];                         # </xxx>
    my $closing = HTML::TagParser::Util::find_closing($flat, $cur);
    my $list    = [];
    while (++$cur < $closing)
      {
        push @$list, $flat->[$cur];
      }
 
    # allow the getElement...() methods on the returned object.
    return bless { flat => $list }, 'HTML::TagParser';
}
 
 
sub nextSibling
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $elem = $flat->[$cur];
 
    return undef if $elem->[000];                         # </xxx>
    my $closing = HTML::TagParser::Util::find_closing($flat, $cur);
    my $next_s = $flat->[$closing+1];
    return undef unless $next_s;
    return undef if $next_s->[000];     # parent's </xxx>
    return HTML::TagParser::Element->new( $flat, $closing+1 );
}
 
sub firstChild
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $elem = $flat->[$cur];
    return undef if $elem->[000];                         # </xxx>
    my $closing = HTML::TagParser::Util::find_closing($flat, $cur);
    return undef if $closing <= $cur+1;                 # no children here.
    return HTML::TagParser::Element->new( $flat, $cur+1 );
}
 
sub childNodes
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $child = firstChild($self);
    return [] unless $child;    # an empty array is easier for our callers than undef
    my @c = ( $child );
    while (defined ($child = nextSibling($child)))
      {
        push @c, $child;
      }
    return \@c;
}
 
sub lastChild
{
    my $c = childNodes(@_);
    return undef unless $c->[0];
    return $c->[-1];
}
 
sub previousSibling
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
 
    ## This one is expensive.
    ## We use find_closing() which walks forward.
    ## We'd need a find_opening() which walks backwards.
    ## So we walk backwards one by one and consult find_closing()
    ## until we find $cur-1 or $cur.
 
    my $idx = $cur-1;
    while ($idx >= 0)
      {
        if ($flat->[$idx][000] && defined($flat->[$idx][006]))
          {
            $idx = $flat->[$idx][006];  # use cache for backwards skipping
            next;
          }
 
        my $closing = HTML::TagParser::Util::find_closing($flat, $idx);
        return HTML::TagParser::Element->new( $flat, $idx )
          if defined $closing and ($closing == $cur || $closing == $cur-1);
        $idx--;
      }
    return undef;
}
 
sub parentNode
{
    my $self = shift;
    my ( $flat, $cur ) = @$self;
 
    return HTML::TagParser::Element->new( $flat, $flat->[$cur][007]) if $flat->[$cur][007];     # cache
 
    ##
    ## This one is very expensive.
    ## We use previousSibling() to walk backwards, and
    ## previousSibling() is expensive.
    ##
    my $ps = $self;
    my $first = $self;
 
    while (defined($ps = previousSibling($ps))) { $first = $ps; }
 
    my $parent = $first->[1] - 1;
    return undef if $parent < 0;
    die "parent too short" if HTML::TagParser::Util::find_closing($flat, $parent) <= $cur;
 
    $flat->[$cur][007] = $parent;       # cache
    return HTML::TagParser::Element->new( $flat, $parent )
}
 
##
## feature:
## self-closing tags have an additional attribute '/' => '/'.
##
sub attributes {
    my $self = shift;
    my ( $flat, $cur ) = @$self;
    my $elem = $flat->[$cur];
    return $elem->[004] if ref $elem->[004];    # cache
    return unless defined $elem->[002];
    my $attr = {};
    while ( $elem->[002] =~ m{
        ([^\s="']+)(\s*=\s*(?:["']((?(?<=")(?:\\"|[^"])*?|(?:\\'|[^'])*?))["']|([^'"\s=]+)['"]*))?
    }sgx ) {
        my $key  = $1;
        my $test = $2;
        my $val  = $3 || $4;
        my $lckey = lc($key);
        if ($test) {
            $key =~ tr/A-Z/a-z/;
            $val = HTML::TagParser::Util::xml_unescape( $val );
            $attr->{$lckey} = $val;
        }
        else {
            $attr->{$lckey} = $key;
        }
    }
    $elem->[004] = $attr;    # cache
    $attr;
}
 
# ----------------------------------------------------------------
 
package HTML::TagParser::Util;
use strict;
 
sub xml_unescape {
    my $str = shift;
    return unless defined $str;
    $str =~ s/&quot;/"/g;
    $str =~ s/&lt;/</g;
    $str =~ s/&gt;/>/g;
    $str =~ s/&amp;/&/g;
    $str;
}
 
sub read_text_file {
    my $file = shift;
    my $fh   = Symbol::gensym();
    open( $fh, $file ) or Carp::croak "$! - $file\n";
    local $/ = undef;
    my $text = <$fh>;
    close($fh);
    $text;
}
 
sub html_to_flat {
    my $txtref = shift;    # reference
    my $flat   = [];
    pos($$txtref) = undef;  # reset matching position
    while ( $$txtref =~ m{
        (?:[^<]*) < (?:
            ( / )? ( [^/!<>\s"'=]+ )
            ( (?:"[^"]*"|'[^']*'|[^"'<>])+ )?
        |
            (!-- .*? -- | ![^\-] .*? )
        ) > ([^<]*)
    }sxg ) {
        #  [000]  $1  close
        #  [001]  $2  tagName
        #  [002]  $3  attributes
        #         $4  comment element
        #  [003]  $5  content
        next if defined $4;
        my $array = [ $1, $2, $3, $5 ];
        $array->[001] =~ tr/A-Z/a-z/;
        #  $array->[003] =~ s/^\s+//s;
        #  $array->[003] =~ s/\s+$//s;
        push( @$flat, $array );
    }
    $flat;
}
 
## returns 1 beyond the end, if not found.
## returns undef if called on a </xxx> closing tag
sub find_closing
{
  my ($flat, $cur) = @_;
 
  return $flat->[$cur][006]        if   $flat->[$cur][006];     # cache
  return $flat->[$cur][006] = $cur if (($flat->[$cur][002]||'') =~ m{/$});    # self-closing
 
  my $name = $flat->[$cur][001];
  #return $flat->[$cur][006] = $cur+1 if $name eq "li";
  my $pre_nest = 0;
  ## count how many levels deep this type of tag is nested.
  my $idx;
  for ($idx = 0; $idx <= $cur; $idx++)
    {
      my $e = $flat->[$idx];
      next unless   $e->[001] eq $name;
      next if     (($e->[002]||'') =~ m{/$});   # self-closing
      $pre_nest += ($e->[000]) ? -1 : 1;
      $pre_nest = 0 if $pre_nest < 0;
      $idx = $e->[006]-1 if !$e->[000] && $e->[006];    # use caches for skipping forward.
    }
  my $last_idx = $#$flat;
 
  ## we move last_idx closer, in case this container
  ## has not all its subcontainers closed properly.
  my $post_nest = 0;
  for ($idx = $last_idx; $idx > $cur; $idx--)
    {
      my $e = $flat->[$idx];
      next unless    $e->[001] eq $name;
      $last_idx = $idx-1;               # remember where a matching tag was
      next if      (($e->[002]||'') =~ m{/$});  # self-closing
      $post_nest -= ($e->[000]) ? -1 : 1;
      $post_nest = 0 if $post_nest < 0;
      last if $pre_nest <= $post_nest;
      $idx = $e->[006]+1 if $e->[000] && defined $e->[006];     # use caches for skipping backwards.
    }
 
  my $nest = 1;         # we know it is not self-closing. start behind.
 
  for ($idx = $cur+1; $idx <= $last_idx; $idx++)
    {
      my $e = $flat->[$idx];
      next unless    $e->[001] eq $name;
      next if      (($e->[002]||'') =~ m{/$});  # self-closing
      if ($name eq "li"){
         # $cur is "li" and so is $idx, so $cur is NOT closed properly - force close
         return $flat->[$cur][006] = $idx;
      }
      $nest      += ($e->[000]) ? -1 : 1;
      if ($nest <= 0)
        {
          die "assert </xxx>" unless $e->[000];
          $e->[006] = $cur;     # point back to opening tag
          return $flat->[$cur][006] = $idx;
        }
      $idx = $e->[006]-1 if !$e->[000] && $e->[006];    # use caches for skipping forward.
    }
 
  # not all closed, but cannot go further
  return $flat->[$cur][006] = $last_idx+1;
}
 
sub find_meta_charset {
    my $txtref = shift;    # reference
    while ( $$txtref =~ m{
        <meta \s ((?: [^>]+\s )? http-equiv\s*=\s*['"]?Content-Type [^>]+ ) >
    }sxgi ) {
        my $args = $1;
        return $1 if ( $args =~ m# charset=['"]?([^'"\s/]+) #sxgi );
    }
    undef;
}
 
sub encode_from_to {
    my ( $txtref, $from, $to ) = @_;
    return     if ( $from     eq "" );
    return     if ( $to       eq "" );
    return $to if ( uc($from) eq uc($to) );
    Encode::from_to( $$txtref, $from, $to, Encode::XMLCREF() );
    return $to;
}
 
# ----------------------------------------------------------------
1;
# ----------------------------------------------------------------
}
#======================================================================