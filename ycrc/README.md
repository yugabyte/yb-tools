# ycrc


ycrc (YCql Row Count) allows counting rows in a large YCQL keyspace by executing parallel queries across smaller partition sizes.



## Usage

```
ycrc <keyspace> [flags]
```


The important flags for optimal performance and to ensure success for sizable datasets are the following:

```
-p --parallel
```
Parallelism - the number of concurrent queries in flight at one time. To a point, increasing this value will decrease the run time of ycrc, in exchange for increased CPU and disk usage on the target cluster.

The default value of 16 is relatively performant for small datasets on small clusters. It's worth experimenting with different parallelism values while tracking CPU usage on a node running tablet server leaders.

In testing, increased parallelism is the number one influence on ycrc performance, up to CPU saturation, assuming scale is set appropriately, after which queries generally fail due to timeouts. That is to say, if CPU usage on a tablet server leader is 20% with a parallelism of `8`, doubling parallelism to `16` will generally half the time ycrc takes to run, at the expense of increasing CPU usage to 40%. This means that one may be able to tune the run of ycrc to enable counting rows while other database activity is occurring, again at a higher time cost, or if increased CPU usage is not too impactful, reduce the runtime by increasing parallelism.




```
-s --scale
```
Scale controls the size of a hash bucket. Low values are performant for tiny to small datasets, but a scale factor which is too small will encounter query timeouts on large datasets.

It is recommended to leave this value at the default of 6, unless you get server-side timeout errors, in which case you should increase this value by 1 until the timeouts are resolved.


```
-t --timeout
```

This is the value, in milliseconds, that the ycrc client allows for a query to respond. The default value of 1500 should be sufficient for most use cases, but increasing to a higher value may be necessary if client side timeouts are noted.



## Building

```
go build
```

Probably required go 1.15, but might not. I don't know, I haven't tested it.

Makefile contribution would be welcome




## Behind the scenes
A query using ycql shell would fail due a timeout after the query runs for too long.

```
select count(*) from mykeyspace.mytable
```

ycrc, however, will instead only execute the count(*) on a slice of the partition:

```
SELECT count(*) as rows from mykeyspace.mytable WHERE partition_hash(id) >= 46608 and partition_hash(id) <= 46623
```

And then sum these up.

The partition size is determined by scale factor, with a scale 1 giving a hash bucket size 512 (that is, `0 <= partition_hash(id) <= 512`) and a scale of 10 giving a hash bucket size 1.

This means that, for every table, scale 1 performs 128 total queries, scanning 512 buckets total, while scale 10 performs 65536 queries.



## Roadmap

This tool will eventually be built and distributed for Windows and Linux, and should have an RPM and possibly other packaging types available for it.
