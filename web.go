package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"sort"
	"strings"
)

var tmpl *template.Template

func startWeb() {
	// load templates
	tmpl = template.New(".")
	template.Must(tmpl.New("dash").Parse(tDash))
	template.Must(tmpl.New("dash-error").Parse(tDashError))
	template.Must(tmpl.New("root").Parse(tRoot))
	template.Must(tmpl.New("info").Parse(tInfo))
	// register handler
	http.HandleFunc("/", handler)
	// start server
	log.Fatal(http.ListenAndServe(config.webUI, nil))
}

type dataDash struct {
	DashData []GraphData
	Path     string
	ListPath string
}

func handler(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/dash") {
		handleDash(w, r)
	} else {
		handleList(w, r)
	}
}

func handleDash(w http.ResponseWriter, r *http.Request) {
	g := r.FormValue("g")
	if len(g) == 0 {
		render(w, "dash-error", nil)
		return
	}
	parts := strings.Split(g, ",")
	td := make([]GraphData, 0, len(parts))
	for i, p := range parts {
		specs := strings.Split(p, "|")
		gd := data.GetDataForGraph(names.FindAll(specs))
		gd.Idx = i
		gd.Title = p
		if pos := strings.Index(gd.Title, "|"); pos > 0 {
			gd.Title = gd.Title[:pos] + "+"
		}
		td = append(td, gd)
	}
	r.URL.RawQuery = ""
	dashPath := "http://" + r.Host + r.URL.String()
	r.URL.Path = r.URL.Path[:len(r.URL.Path)-5] // ends with "/dash"
	listPath := "http://" + r.Host + r.URL.String()
	data := dataDash{
		DashData: td,
		Path:     dashPath,
		ListPath: listPath,
	}
	render(w, "dash", data)
}

type dataList struct {
	Counters []string
	Timers   []string
	Gauges   []string
	Sets     []string
	Path     string
	Empty    bool
	Config   string
	Mem      string
}

func handleList(w http.ResponseWriter, r *http.Request) {
	all := names.List()
	data := dataList{
		Counters: all[0],
		Timers:   all[1],
		Gauges:   all[2],
		Sets:     all[3],
		Empty:    len(all[0])+len(all[1])+len(all[2])+len(all[3]) == 0,
	}
	sort.Strings(data.Counters)
	sort.Strings(data.Timers)
	sort.Strings(data.Gauges)
	sort.Strings(data.Sets)
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	data.Mem = fmt.Sprintf("resource usage: %.2f MiB heap, %.2f MiB sysvm, %d goroutines",
		float64(stats.Alloc)/1048576, float64(stats.Sys)/1048576,
		runtime.NumGoroutine())
	data.Config = fmt.Sprintf("config: flush interval %v, retention %v, percentiles %v",
		config.flush, config.retention, config.percentiles)
	r.URL.RawQuery = ""
	if !strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path += "/"
	}
	r.URL.Path = r.URL.Path + "dash"
	data.Path = "http://" + r.Host + r.URL.String()
	render(w, "root", data)
}

func render(w http.ResponseWriter, tname string, data interface{}) {
	if err := tmpl.ExecuteTemplate(w, tname, data); err != nil {
		log.Print(err)
	}
}

const tDashError = `
You have an error in your query.
`

const tDash = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>statsd-viz Dashboard</title>
    <link href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css" rel="stylesheet">
	<link href='https://fonts.googleapis.com/css?family=Source+Sans+Pro' rel='stylesheet' type='text/css'>
	<style type="text/css">
	body { background-color: #f8f8f8; font-family: "Source Sans Pro", sans-serif; font-size: 12px; }
	.chart {
		width: 324px; height: 200px; border-radius: 3px; background-color: #fff;
		box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24);
		margin: 5px;
	}
	.chartc {
		display: flex; display: -webkit-flex; flex-wrap: wrap; -webkit-flex-wrap: wrap;
	}
	.dygraph-legend {
	  font-size: 12px !important;background: #f7ca88 !important; padding: 2px;
	  margin-top: 158px; z-index: 20 !important; left: 52px !important;
    }
	.dygraph-title { font-size: 14px; font-weight: 400; }
	h2 { text-align: center; font-size: 24px; padding: 1.1em; }
	.footer { margin: 4em 0 2em 0; color: #999; text-align: center; font-size: 14px }
	</style>
  </head>
  <body>
  	<div class="container-fluid">
	  <div class="row">
	    <div class="col-sm-12">
			<h2>statsd-vis • dashboard</h2>
		</div>
	  </div>
	  <div class="row">
	    <div class="col-sm-12 chartc">
		  {{range .DashData}}
		  <div id="id-{{.Idx}}" class="chart"></div>
		  {{end}}
		</div>
	  </div>
	  <div class="row" style="padding-top: 4em; font-size: 18px; text-align: center">
	    [ <a href="{{.ListPath}}">metrics list</a> ]
	  </div>
	  {{template "info" .}}
	  <div class="row footer">
		<a href="https://statsd-vis.info">statsd-vis</a> &mdash; &copy; 2017 <a href="https://www.rapidloop.com/">RapidLoop</a>
		<br>
		If you like this, you might like <a href="https://www.opsdash.com">OpsDash</a> - easy-to-use server, infra and app monitoring.
	  </div>
	</div>

    <script src="https://code.jquery.com/jquery-2.1.4.min.js"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/js/bootstrap.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/dygraph/1.1.1/dygraph-combined.js"></script>
	<script type="text/javascript">
	$(function() {
	  {{range .DashData}}
		new Dygraph(
		  document.getElementById("id-{{.Idx}}"),
		  [
			{{range .Datapoints}}
			[ new Date( {{.At.Unix}} * 1000 ), {{.ValuesStr}} ],
			{{end}}
		  ],
		  {
			title: "{{.Title}}",
			axisLabelFontSize: 10,
			axes: { y: { axisLabelWidth: 30 } },
			labels: [ "X", {{range .Metrics}}"{{.}}",{{end}} ],
      		gridLineColor: 'rgb(200,200,200)',
			labelsSeparateLines: true,
			connectSeparatedPoints: true,
			//drawPoints: true
		  }
		);
	  {{end}}
	  var search = location.search || '';
	  if (search.indexOf('refresh') !== -1) {
	    window.setTimeout(function() {
	      window.location.reload();
	    }, 10000);
	  }
	});
	</script>
  </body>
</html>
`

const tRoot = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>statsd-viz</title>
    <link href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css" rel="stylesheet">
	<link href='https://fonts.googleapis.com/css?family=Source+Sans+Pro' rel='stylesheet' type='text/css'>
	<link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
	<style type="text/css">
	body { background: #f8f8f8; color: #383838; font-family: "Source Sans Pro", sans-serif; font-size: 16px; }
	h2 { text-align: center; font-size: 24px; padding: 1.1em; }
	.footer { margin: 5em 0 2em 0; color: #999; text-align: center; font-size: 14px }
	.names { font-size: 16px; line-height: 1.5em; padding-top: 1.1em; }
	.col1, .col1 a {color: #7cafc2}
	.col2, .col2 a {color: #a1b56c}
	.col3, .col3 a {color: #ba8baf}
	.col4, .col4 a {color: #dc9656}
	.head { background-color: #e8e8e8; font-size: 18px }
	.head div { display: inline-block; vertical-align: super; padding-top: 4px }
	</style>
  </head>
  <body>
  	<div class="container-fluid">
	  <div class="row">
	    <div class="col-sm-12">
			<h2>statsd-vis • metrics list</h2>
		</div>
	  </div>
	  <div class="row head">
	  	<div class="col-sm-3"><i class="material-icons col1">plus_one</i> <div>Counters</div></div>
	  	<div class="col-sm-3"><i class="material-icons col2">timer</i> <div>Timers</div></div>
	  	<div class="col-sm-3"><i class="material-icons col3">equalizer</i> <div>Gauges</div></div>
	  	<div class="col-sm-3"><i class="material-icons col4">view_module</i> <div>Sets</div></div>
	  </div>
	  {{if .Empty}}
	  <div class="row" style="padding-top: 2em; text-align: center">
		No metrics yet. Once you start sending in your metrics to the
		StatsD port, they will be listed here.
	  </div>
	  {{else}}
	  <div class="row names">
	    <div class="col-sm-3 col1">
		{{$path := .Path}}
		{{range .Counters}}
		<a href="{{$path}}?g={{.}}">{{.}}</a><br>
		{{end}}
		</div>
	    <div class="col-sm-3 col2">
		{{range .Timers}}
		<a href="{{$path}}?g={{.}}">{{.}}</a><br>
		{{end}}
		</div>
	    <div class="col-sm-3 col3">
		{{range .Gauges}}
		<a href="{{$path}}?g={{.}}">{{.}}</a><br>
		{{end}}
		</div>
	    <div class="col-sm-3 col4">
		{{range .Sets}}
		<a href="{{$path}}?g={{.}}">{{.}}</a><br>
		{{end}}
		</div>
	  </div>
	  {{end}}
	  {{template "info" .}}
	  <div class="row footer">
	    <p>
	    {{.Config}}
		<br>
	    {{.Mem}}
	    <p>
		<a href="https://statsd-vis.info">statsd-vis</a> &mdash; &copy; 2017 <a href="https://www.rapidloop.com/">RapidLoop</a>
		<br>
		If you like this, you might like <a href="https://www.opsdash.com">OpsDash</a> - easy-to-use server, infra and app monitoring.
	  </div>
	</div>

    <script src="https://code.jquery.com/jquery-2.1.4.min.js"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/js/bootstrap.min.js"></script>
  </body>
</html>
`

const tInfo = `
<div class="row" style="padding-top: 4em; font-size: 14px">
  <div class="col-sm-8 col-sm-offset-2">
    <div class="panel panel-default" style="color: #484848; background-color: transparent">
      <div class="panel-body" style="padding: 1.3em">
<p>
To view the graph for a metric M, simply click on it or navigate to <a href="{{.Path}}?g=M">{{.Path}}?g=M</a>
<p>
To view metrics M<sub>1</sub>, M<sub>2</sub> etc. as different graphs, use
<a href="{{.Path}}?g=M1,M2">{{.Path}}?g=M1,M2</a>
<p>
To view M<sub>1</sub> and M<sub>2</sub> in one graph and M<sub>3</sub> and
M<sub>4</sub> in another, use
<a href="{{.Path}}?g=M1|M2,M3|M4">{{.Path}}?g=M1|M2,M3|M4</a>
<p>
M matches an initial prefix of the actual metric name. This is helpful
when using timers &ndash; so "my.timer" will also match the generated metric names
"my.timer.lower", "my.timer.upper_95" etc.
<p style="margin: 0">
Append "&amp;refresh" to let the page reload itself every minute, like this: <a href="{{.Path}}?g=M&refresh">{{.Path}}?g=M&refresh</a>
      </div>
    </div>
  </div>
</div>
`
