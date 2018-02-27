
# statsd-vis

_statsd-vis_ is a standalone, zero-dependency single-binary
[StatsD](https://github.com/etsy/statsd) server with built-in web UI
with which you can visualize graphs.

It holds time series data for a configurable time in-memory, and does not
persist or forward it.

## build

statsd-vis is written entirely in [Go](https://golang.org/). To build it,
you can `go get` it:

    go get github.com/rapidloop/statsd-vis

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

There is also a [statsd-vis Docker image](https://hub.docker.com/r/rapidloop/statsd-vis/)
on Docker Hub, built from source on each commit. You can run the Docker image with:

    docker run --rm -it -p 8080:8080 -p 8125:8125/udp -p 8125:8125/tcp rapidloop/statsd-vis -statsdudp 0.0.0.0:8125 -statsdtcp 0.0.0.0:8125

Notice that for Docker usage, you have to listen on 0.0.0.0, since the default 127.0.0.1 won't be reachable from outside the container, even from the Docker host.

## changelog

* v0.1, 13-May-2017: first public release
