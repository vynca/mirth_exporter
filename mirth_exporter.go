package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const namespace = "mirth"

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last Mirth query successful.",
		nil, nil,
	)
	channelsDeployed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "channels_deployed"),
		"How many channels are deployed.",
		nil, nil,
	)
	channelsStarted = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "channels_started"),
		"How many of the deployed channels are started.",
		nil, nil,
	)
	messagesReceived = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_received_total"),
		"How many messages have been received (per channel).",
		[]string{"channel"}, nil,
	)
	messagesFiltered = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_filtered_total"),
		"How many messages have been filtered (per channel).",
		[]string{"channel"}, nil,
	)
	messagesQueued = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_queued"),
		"How many messages are currently queued (per channel).",
		[]string{"channel"}, nil,
	)
	messagesSent = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_sent_total"),
		"How many messages have been sent (per channel).",
		[]string{"channel"}, nil,
	)
	messagesErrored = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_errored_total"),
		"How many messages have errored (per channel).",
		[]string{"channel"}, nil,
	)
)

type Exporter struct {
	jarPath, configPath string
}

func NewExporter(mccliJarPath, mccliConfigPath string) *Exporter {
	return &Exporter{
		jarPath:    mccliJarPath,
		configPath: mccliConfigPath,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- channelsDeployed
	ch <- channelsStarted
	ch <- messagesReceived
	ch <- messagesFiltered
	ch <- messagesQueued
	ch <- messagesSent
	ch <- messagesErrored
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	lines, err := e.fetchStatLines()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Error(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)
	e.readStatus(lines, ch)
	e.readChannelStats(lines, ch)
}

func (e *Exporter) fetchStatLines() ([]string, error) {
	cmd := exec.Command("java", "-jar", e.jarPath, "-c", e.configPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	fmt.Fprintln(stdin, "status")
	fmt.Fprintln(stdin, "channel stats")
	stdin.Close()
	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	lines := strings.Split(string(bytesOut), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("Unexpected output: %s", string(bytesOut))
	}
	log.Debug(string(bytesOut))
	return lines, nil
}

func (e *Exporter) readStatus(lines []string, ch chan<- prometheus.Metric) {
	deployed := regexp.MustCompile(`^[0-9a-f-]{36}\s+[a-zA-Z]+\s+`)
	started := regexp.MustCompile(`\s+Started\s+`)
	deployedCount, startedCount := 0, 0
	for _, line := range lines {
		if deployed.MatchString(line) {
			deployedCount++
			if started.MatchString(line) {
				startedCount++
			}
		}
	}
	ch <- prometheus.MustNewConstMetric(
		channelsDeployed, prometheus.GaugeValue, float64(deployedCount),
	)
	ch <- prometheus.MustNewConstMetric(
		channelsStarted, prometheus.GaugeValue, float64(startedCount),
	)
}

func (e *Exporter) readChannelStats(lines []string, ch chan<- prometheus.Metric) {
	stat := regexp.MustCompile(`^(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(.+)$`)
	for _, line := range lines {
		if stat.MatchString(line) {
			group := stat.FindStringSubmatch(line)
			channel := group[6]
			received, _ := strconv.ParseFloat(group[1], 64)
			ch <- prometheus.MustNewConstMetric(
				messagesReceived, prometheus.CounterValue, received, channel,
			)
			filtered, _ := strconv.ParseFloat(group[2], 64)
			ch <- prometheus.MustNewConstMetric(
				messagesFiltered, prometheus.CounterValue, filtered, channel,
			)
			queued, _ := strconv.ParseFloat(group[3], 64)
			ch <- prometheus.MustNewConstMetric(
				messagesQueued, prometheus.GaugeValue, queued, channel,
			)
			sent, _ := strconv.ParseFloat(group[4], 64)
			ch <- prometheus.MustNewConstMetric(
				messagesSent, prometheus.CounterValue, sent, channel,
			)
			errored, _ := strconv.ParseFloat(group[5], 64)
			ch <- prometheus.MustNewConstMetric(
				messagesErrored, prometheus.CounterValue, errored, channel,
			)
		}
	}
}

func main() {
	var (
		listenAddress = flag.String("web.listen-address", ":9140",
			"Address to listen on for telemetry")
		metricsPath = flag.String("web.telemetry-path", "/metrics",
			"Path under which to expose metrics")
		mccliConfigPath = flag.String("mccli.config-path", "./mirth-cli-config.properties",
			"Path to properties file for Mirth Connect CLI")
		mccliJarPath = flag.String("mccli.jar-path", "./mirth-cli-launcher.jar",
			"Path to jar file for Mirth Connect CLI")
	)
	flag.Parse()

	exporter := NewExporter(*mccliJarPath, *mccliConfigPath)
	prometheus.MustRegister(exporter)

	log.Infof("Starting server: %s", *listenAddress)
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Mirth Exporter</title></head>
             <body>
             <h1>Mirth Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
