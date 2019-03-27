// Copyright (C) 2016, 2017 Nicolas Lamirault <nicolas.lamirault@gmail.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/dchest/uniuri"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const (
	version   = "0.2.0"
	namespace = "speedtest"
)

var (
	defaultConfigURL     = "http://c.speedtest.net/speedtest-config.php?x=" + uniuri.New()
	defaultServerListURL = "http://c.speedtest.net/speedtest-servers-static.php?x=" + uniuri.New()

	server = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "server",
		Help:      "Server details",
	}, []string{"name", "latitude", "longtitude"})
	latency = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "latency",
		Help:      "Latency (seconds)",
	})
	upload = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "upload",
		Help:      "Upload bandwidth (bit/s)",
	})
	download = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "download",
		Help:      "Download bandwidth (bit/s)",
	})
)

// Exporter collects Speedtest stats from the given server and exports them using
// the prometheus metrics package.
func SpeedtestMiddleware(c *Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metric := c.NetworkMetrics()
		server.With(prometheus.Labels{
			"name":       metric.Server.Name,
			"latitude":   fmt.Sprintf("%f", metric.Server.Lat),
			"longtitude": fmt.Sprintf("%f", metric.Server.Lon)}).Set(1)

		latency.Set(metric.Latency)
		upload.Set(metric.UploadSpeed)
		download.Set(metric.DownloadSpeed)

		next.ServeHTTP(w, r)
	})
}

func init() {
	prometheus.MustRegister(server)
	prometheus.MustRegister(latency)
	prometheus.MustRegister(upload)
	prometheus.MustRegister(download)
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9112", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		configURL     = flag.String("speedtest.config-url", defaultConfigURL, "Speedtest configuration URL")
		serverListURL = flag.String("speedtest.server-list-url", defaultServerListURL, "Speedtest server list URL")
		reloadServer  = flag.Bool("speedtest.reload-server", false, "Always try and find the fastest server before each test")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Speedtest Prometheus exporter. v%s\n", version)
		os.Exit(0)
	}

	client, err := NewClient(*configURL, *serverListURL, *reloadServer)
	if err != nil {
		log.Fatalln("Can't create exporter:", err)
	}

	http.Handle(*metricsPath, SpeedtestMiddleware(client, promhttp.Handler()))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Speedtest Exporter</title></head>
             <body>
             <h1>Speedtest Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
