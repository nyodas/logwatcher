
# Logwatcher [![Build Status](https://travis-ci.org/nyodas/logwatcher.svg?branch=master)](https://travis-ci.org/nyodas/logwatcher)

# How to build
```
go get github.com/nyodas/logwatcher
```
```bash
# Restore vendor just in case.
which dep || go get -u github.com/golang/dep/cmd/dep 
dep ensure

go build
```

# How to run
```
# Termui output (pretty)
# Use the k & j key to move between metrics pane and alerts history.
# Use ctrl+c or q to quit
./logwatcher -f /tb9/opt/nginx/log/access.log -i 10s --ui
# Standard console output (less pretty ,history w/h scrolling)
./logwatcher -f /tb9/opt/nginx/log/access.log -i 10s

usage: logwatcher --file=FILE [<flags>]
Flags:
      --help         Show context-sensitive help (also try --help-long and --help-man).
  -i, --interval=10s Interval for stats logging in seconds
  -a, --alert=10     Ceiling for the alert.
  -f, --file=FILE    File to watch
  -u, --ui           Use termui
      --version      Show application version.
```
## What it looks like
![Pane1-metrics](https://raw.githubusercontent.com/nyodas/logwatcher/master/docs/standard.png)
![Pane2-history](https://raw.githubusercontent.com/nyodas/logwatcher/master/docs/alerthistory.png)

### Status
- [X] Follow logs
    - [X] Survive log rotate
- [x] Understand NCSA format
- [x] Console app. (Don't die)
- [X] Log every X time the tops path
    - [ ] Add interesting summary stats
        - [ ] Body size
        - [X] Total Request Made
        - [ ] Users
        - [ ] HTTP method
        - [ ] HTTP Code (Error Rate)
- [X] Alerting on high traffic. (X interval)
- [X] Recover on lower traffic. (X interval)
- [X] History of alert on traffic.
- [X] Unit Test
- [ ] More Unit Test
- [X] Clean up
- [ ] More Clean up
- [ ] Smarter and clean Code w/h interface and so on

### Improvement on project
 - [X] Add flags to provide customisation on interval.
 - [ ] Add flags to parse an historical log file.
 - [ ] [Termui]('https://github.com/gizak/termui') - Proper layout for printing stats and posibly graph.
     - [X] Started work in #termui branch 
     - [X] Make it availlable via a flag
 - [ ] [Prometheus]('https://github.com/prometheus/prometheus')/[Timeseries]('https://github.com/prometheus/tsdb') in app
    - [ ] Alerting and query done by smarter men than me.
    - [ ] More Accuracy on metrics
    - [X] In-between could be [go-metrics]('https://github.com/rcrowley/go-metrics')
    - [ ] Tried and failed for now to make it work #promTsdbImplem branch
 - [ ] Exporting said metrics to a remote metrics system (Prometheus,Datadog)
 - [ ] Extend the facility to parse more log format (nginx combined & more)
 - [ ] More alerting options and more alerting rules.
 - [ ] Have a web page for easy historical data.
 - [ ] Fix all the TODO
