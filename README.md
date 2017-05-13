
# statsd-vis

_statsd-vis_ is a standalone, zero-dependency single-binary
[StatsD](https://github.com/etsy/statsd) server with built-in web UI
with which you can visualize graphs.

It holds time series data for a configurable time in-memory, and does not
persist or forward it.

*statsd-vis's [home page](https://statsd-vis.info) has more
information and a live demo!*

## build

statsd-vis is written entirely in [Go](https://golang.org/). To build it,
you can `go get` it:

    go get github.com/rapidloop/statd-vis

You should find the binary `statsd-vis` under `$GOPATH/bin` when the command
completes. There are no runtime dependencies or configuration needed.

## command-line

You can set parameters like the flush interval, percentiles etc. on the
command-line:

```
statsd-vis 0.1 - (c) 2017 RapidLoop - MIT Licensed - https://statsd-vis.info/
statd-vis is a standalone statsd server with built-in visualization

  -flush interval
    	flush interval (default 10s)
  -percentiles string
    	percentiles for timer metrics (default "90,95,99")
  -retention duration
    	duration to retain the metrics for (default 30m0s)
  -statsdtcp address
    	statsd TCP listen address (default "127.0.0.1:8125")
  -statsdudp address
    	statsd UDP listen address (default "127.0.0.1:8125")
  -webui address
    	web UI listen address (default "0.0.0.0:8080")
```

## releases

You can get pre-built binaries for releases from the
[releases page](https://github.com/rapidloop/statsd-vis/releases).

## changelog

* v0.1, 13-May-2017: first public release
