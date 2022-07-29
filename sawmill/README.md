# sawmill

Log processor for YugabyteDB logs

## Description

This program takes multiple YugabyteDB logs, slices them up, sorts them, and outputs a single log file with a monotonically increasing timestamp.  It allows you to see a timeline of all cluster activity in a single file.

Inputs are tserver and master log files.  Output goes to stdout, so you will need to redirect the output to a file.

For example, let's say you have two log files from different hosts that look like this:

```
Log file created at: 2022/07/26 18:30:24
Running on machine: yb-dev-load-n1
Application fingerprint: version 2.13.3.0 build PRE_RELEASE revision e97adaf95bfa54f4f82993dbe868d094e036ddc2 build_type RELEASE built at 13 Jun 2022 18:52:35 UTC
Running duration (h:mm:ss): 0:00:00
Log line format: [IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg
W0726 18:30:24.487058  7466 catalog_manager.cc:1318] Failed to get current config: Illegal state (yb/master/catalog_manager.cc:10300): Node 6ba1ab057e4b4caaa46aea0e211a41f3 peer not initialized.
```

and this:

```
Log file created at: 2022/07/26 18:30:26
Running on machine: yb-dev-load-n2
Application fingerprint: version 2.13.3.0 build PRE_RELEASE revision e97adaf95bfa54f4f82993dbe868d094e036ddc2 build_type RELEASE built at 13 Jun 2022 18:52:35 UTC
Running duration (h:mm:ss): 0:00:00
Log line format: [IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg
E0726 18:30:26.041532  6295 async_initializer.cc:95] Failed to initialize client: Timed out (yb/rpc/rpc.cc:224): Could not locate the leader master: GetLeaderMasterRpc(addrs: [10.6.0.10:7100, 10.6.0.12:7100, 10.6.0.8:7100], num_attempts: 53) passed its deadline 18699.722s (passed: 1.542s): Not found (yb/master/master_rpc.cc:286): no leader found: GetLeaderMasterRpc(addrs: [10.6.0.10:7100, 10.6.0.12:7100, 10.6.0.8:7100], num_attempts: 1)
```

You can run `sawmill log1 log2` to get the following output:

```
W0726 18:30:24.487058 <yb-dev-load-n1>  7466 catalog_manager.cc:1318] Failed to get current config: Illegal state (yb/master/catalog_manager.cc:10300): Node 6ba1ab057e4b4caaa46aea0e211a41f3 peer not initialized.
E0726 18:30:26.041532 <yb-dev-load-n2>  6295 async_initializer.cc:95] Failed to initialize client: Timed out (yb/rpc/rpc.cc:224): Could not locate the leader master: GetLeaderMasterRpc(addrs: [10.6.0.10:7100, 10.6.0.12:7100, 10.6.0.8:7100], num_attempts: 53) passed its deadline 18699.722s (passed: 1.542s): Not found (yb/master/master_rpc.cc:286): no leader found: GetLeaderMasterRpc(addrs: [10.6.0.10:7100, 10.6.0.12:7100, 10.6.0.8:7100], num_attempts: 1)
```

## Usage

Run it on a support package like this:

```
sawmill master/logs/*.log.[EIW]*[^z] tserver/logs/*.log.[EIW]*[^z] > sorted
```

There is also a `-v` option that works like `egrep -v` where you can filter regular expressions from the output.  For example, if you want to ignore certain strings in the generated output, you can use:

```
sawmill -v '(ignore1|ignore2|ignore3)' *.log > sorted
```

## Build

```
make
```

You'll need XCode on macOS, and Ubuntu requires `flex` (and `flex-devel` on CentOS) packages installed.

## zebra

There's an optional helper script named `zebra` which will print files with a blank line between each line to make it easier to read.

