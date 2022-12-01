#!/usr/bin/env perl
# heatseeker: find hot shards by aggregating disk usage by tablet by node
# parses output of 'yugatool cluster_info --tablet-report'

@units = ('b', 'kb', 'mb', 'gb', 'tb');
@exponents = 0 .. $#units;
@exponent{@units} = @exponents;

while (<>) {
	# skip the beginning sections and empty lines
	next if (m/^$/);
	if (m/Tablet Report: \[host:"([^"]+)/) {
		$host = $1;
		$seen_it = 1;
		next;
	}
	next unless $seen_it;

	($tablet, $table, $namespace, $state, $status, $start_key, $end_key, $sst_size, $sst_units, $wal_size, $wal_units, $cterm, $cidx, $leader, $lease_status) = split(' ');
	next if ($tablet eq "TABLET"); # skip header line

	$summation{$table}{"$tablet,$host"} += ($sst_size * 1024**$exponent{lc $sst_units}) + ($wal_size * 1024**$exponent{lc $wal_units});
}

foreach $table (keys %summation) {
	foreach $tuple (keys %{$summation{$table}}) {
		# build an array of sums for each table/tablet corresponding to each host
		$size = $summation{$table}{$tuple};
		($tablet, $host) = split(/,/, $tuple);
		push(@{$report{$table}{$tablet}}, $size);
		$digits = $size > 0 ? int(log($size)/log(10))+1 : 0;
		$digits{$table} = $digits if ($digits > $digits{$table}); # max()
	}
}

foreach $table (keys %summation) {
	print "$table (" . scalar(keys %{$report{$table}}) . " shards)\n"; 
	foreach $tablet (keys %{$report{$table}}) {
		print "$tablet ";
		@size = @{$report{$table}{$tablet}};
		$format = sprintf "%%%sd, ", $digits{$table};
		printf "$format", $_ for @size[0..$#size-1];
		$format =~ s/,.*//;
		printf "$format\n", @size[$#size];
	}
}
