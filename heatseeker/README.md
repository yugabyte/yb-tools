# heatseeker

Find hot shards

## Description

This program helps find hot shards by noting imbalances in shard disk utilization.  It tallies the disk usage per table, per tablet, per host, and then lists all the tablets for each table.

The following example output shows a normal table with 12 shards, each approximately equal in size in terms of bytes used on disk.  The trailing three columns indicate a replication factor of 3, and they show relatively equal disk usage across the three replicas.

```
table-000033e500003000800000000000641f (12 shards)
tablet-dce67ac6f5034ed492f039b552da23b6 17396, 17400, 17400
tablet-432654ded417478b857bcb06e9be511c 17392, 17400, 17400
tablet-70eeb747d1ec484cb38d6d40a440c814 17400, 17400, 17400
tablet-2cc3bde4544449a6a7ad53171cdd52ac 17400, 17400, 17400
tablet-9530b4482f2f4a8fb1cf80df72053495 17396, 17396, 17396
tablet-8fe131884479471fa2951247d2f70c39 17400, 17400, 17400
tablet-793ad9711f60456b91d155975541e60e 17400, 17396, 17400
tablet-c3d569ab179f49afb67b86f06b917ade 17396, 17400, 17400
tablet-0b52f6151068421b9411af0a2cbcc47a 17396, 17400, 17400
tablet-49c5feef67764dc789d171bf181a5ee5 17400, 17396, 17392
tablet-d2ecd2971822426abc9cc2e1fce8e669 17400, 17400, 17396
tablet-54a752a9469949eeba37873f271478a5 17396, 17400, 17396
```

Here is an example of a hot shard.  Notice how one shard jumps out as being orders of magnitude bigger than the rest:

```
table-000080b9000030008000000000008ec5
tablet-666ecacbd9104403abb2634057e51558 17708, 17708, 17712
tablet-aad2154d06484c56bc8336e0d7b0182b 17708, 17708, 17704
tablet-f927430690094146a2b9027ab9944de4 17708, 17708, 17704
tablet-5faedc265efd4c8b9a07648f2053c654 17708, 17704, 17708
tablet-6fe4ce0f91bf4029887c6f2f01c9d1e1 17704, 17708, 17708
tablet-493d78c48e6e40b5af219a6ad39ba798 21080122, 21352737, 21296320
```

## Running

The first step is gathering the telemetry from all the nodes in the cluster using a helper script named `gettablets.sh`.  This script will ssh to each node, run a command that outputs the disk usage in a specific format, and then create a file named `data` with the IP address of the host as the suffix.  It assumes the ssh key resides in `$HOME/pem`.  An example invocation looks like this:

```
./gettablets.sh 10.6.0.10 10.6.0.11 10.6.0.12 10.6.0.6 10.6.0.7 10.6.0.8
```

Since it can be tedious to type in IP addresses, there is yet another helper script which will take the endpoints copied from the platform GUI's Connect button and turn them into a space-separated list.

```
./connect2space 10.6.0.10:5433,10.6.0.11:5433,10.6.0.12:5433,10.6.0.6:5433,10.6.0.7:5433,10.6.0.8:5433
10.6.0.10 10.6.0.11 10.6.0.12 10.6.0.6 10.6.0.7 10.6.0.8
```

You can then use the output of the helper script as arguments to `gettablets.sh`.  Doing this inline would look like this:

```
./gettablets.sh `./connect2space 10.6.0.10:5433,10.6.0.11:5433,10.6.0.12:5433,10.6.0.6:5433,10.6.0.7:5433,10.6.0.8:5433`
```

Once you have the `data.*` files, run the main tool to generate output:


```
./htskr data.*
```

This will create a report you can peruse to see if any shards jump out as being bigger than the rest.