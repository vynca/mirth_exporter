package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

const namespace = "mirth"

type Exporter struct {
	sync.Mutex
	jarPath, configPath string
	statLines           []string
	up                  prometheus.Gauge
	channelsDeployed    prometheus.Gauge
	channelsStarted     prometheus.Gauge
	messagesReceived    *prometheus.CounterVec
	messagesFiltered    *prometheus.CounterVec
	messagesQueued      *prometheus.GaugeVec
	messagesSent        *prometheus.CounterVec
	messagesErrored     *prometheus.CounterVec
}

func NewExporter(mccliJarPath, mccliConfigPath *string) *Exporter {
	return &Exporter{
		jarPath:    *mccliJarPath,
		configPath: *mccliConfigPath,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last Mirth query successful.",
		}),
		channelsDeployed: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "channels_deployed",
			Help:      "How many channels are deployed.",
		}),
		channelsStarted: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "channels_started",
			Help:      "How many of the deployed channels are started.",
		}),
		messagesReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_received",
				Help:      "How many messages have been received (per channel).",
			},
			[]string{"channel"},
		),
		messagesFiltered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_filtered",
				Help:      "How many messages have been filtered (per channel).",
			},
			[]string{"channel"},
		),
		messagesQueued: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "messages_queued",
				Help:      "How many messages are currently queued (per channel).",
			},
			[]string{"channel"},
		),
		messagesSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_sent",
				Help:      "How many messages have been sent (per channel).",
			},
			[]string{"channel"},
		),
		messagesErrored: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_errored",
				Help:      "How many messages have errored (per channel).",
			},
			[]string{"channel"},
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.channelsDeployed.Desc()
	ch <- e.channelsStarted.Desc()
	e.messagesReceived.Describe(ch)
	e.messagesFiltered.Describe(ch)
	e.messagesQueued.Describe(ch)
	e.messagesSent.Describe(ch)
	e.messagesErrored.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()
	err := e.collect()
	if err != nil {
		e.up.Set(0)
		ch <- e.up
		log.Error(err)
		return
	}
	e.up.Set(1)
	ch <- e.up
	ch <- e.channelsDeployed
	ch <- e.channelsStarted
	e.messagesReceived.Collect(ch)
	e.messagesFiltered.Collect(ch)
	e.messagesQueued.Collect(ch)
	e.messagesSent.Collect(ch)
	e.messagesErrored.Collect(ch)
}

func (e *Exporter) collect() error {
	err := e.fetchStatLines()
	if err != nil {
		return err
	}
	e.parseStatus()
	e.parseChannelStats()
	return nil
}

func (e *Exporter) fetchStatLines() error {
	cmd := exec.Command("java", "-jar", e.jarPath, "-c", e.configPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprintln(stdin, "status")
	fmt.Fprintln(stdin, "channel stats")
	stdin.Close()
	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	e.statLines = strings.Split(string(bytesOut), "\n")
	if len(e.statLines) < 3 {
		return fmt.Errorf("Unexpected output: %s", string(bytesOut))
	}
	log.Debug(string(bytesOut))
	return nil
}

func (e *Exporter) parseStatus() {
	deployed := regexp.MustCompile(`^[0-9a-f-]{36}\s+[a-zA-Z]+\s+`)
	started := regexp.MustCompile(`\s+Started\s+`)
	deployedCount, startedCount := 0, 0
	for _, line := range e.statLines {
		if deployed.MatchString(line) {
			deployedCount++
			if started.MatchString(line) {
				startedCount++
			}
		}
	}
	e.channelsDeployed.Set(float64(deployedCount))
	e.channelsStarted.Set(float64(startedCount))
}

func (e *Exporter) parseChannelStats() {
	stat := regexp.MustCompile(`^(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(.+)$`)
	for _, line := range e.statLines {
		if stat.MatchString(line) {
			group := stat.FindStringSubmatch(line)
			channel := group[6]
			received, _ := strconv.ParseFloat(group[1], 64)
			e.messagesReceived.WithLabelValues(channel).Set(received)
			filtered, _ := strconv.ParseFloat(group[2], 64)
			e.messagesFiltered.WithLabelValues(channel).Set(filtered)
			queued, _ := strconv.ParseFloat(group[3], 64)
			e.messagesQueued.WithLabelValues(channel).Set(queued)
			sent, _ := strconv.ParseFloat(group[4], 64)
			e.messagesSent.WithLabelValues(channel).Set(sent)
			errored, _ := strconv.ParseFloat(group[5], 64)
			e.messagesErrored.WithLabelValues(channel).Set(errored)
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

	exporter := NewExporter(mccliJarPath, mccliConfigPath)
	prometheus.MustRegister(exporter)

	log.Infof("Starting server: %s", *listenAddress)
	http.Handle(*metricsPath, prometheus.Handler())
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
