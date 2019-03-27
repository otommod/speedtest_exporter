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
	"github.com/prometheus/common/log"
	"github.com/zpeters/speedtest/print"
	"github.com/zpeters/speedtest/sthttp"
	"github.com/zpeters/speedtest/tests"
)

const (
	userAgent = "speedtest_exporter"
)

type SpeedtestResults struct {
	Server        sthttp.Server
	DownloadSpeed float64
	UploadSpeed   float64
	Latency       float64
}

// Client defines the Speedtest client
type Client struct {
	reloadServer    bool
	Server          sthttp.Server
	AllServers      []sthttp.Server
	SpeedtestClient *sthttp.Client
}

// NewClient defines a new client for Speedtest
func NewClient(configURL string, serverListURL string, reloadServer bool) (*Client, error) {
	stClient := sthttp.NewClient(
		&sthttp.SpeedtestConfig{
			ConfigURL:       configURL,
			ServersURL:      serverListURL,
			AlgoType:        "max",
			NumClosest:      3,
			NumLatencyTests: 5,
			Interface:       "",
			Blacklist:       []string{},
			UserAgent:       userAgent,
		},
		&sthttp.HTTPConfig{},
		true,
		"|")

	log.Debug("Retrieve configuration")
	config, err := stClient.GetConfig()
	if err != nil {
		return nil, err
	}
	stClient.Config = &config

	print.EnvironmentReport(stClient)

	log.Debug("Retrieve all servers")
	allServers, err := stClient.GetServers()
	if err != nil {
		return nil, err
	}

	stClient.GetClosestServers(allServers)
	return &Client{
		reloadServer:    reloadServer,
		SpeedtestClient: stClient,
		AllServers:      allServers,
	}, nil
}

func (client *Client) NetworkMetrics() SpeedtestResults {
	var zeroServer sthttp.Server
	if client.reloadServer || client.Server == zeroServer {
		client.Server = client.SpeedtestClient.GetFastestServer(client.AllServers)
		log.Infoln("Test server:", client.Server)
	}

	// XXX: tester may os.Exit in case of errors; we can't catch that
	tester := tests.NewTester(client.SpeedtestClient, tests.DefaultDLSizes, tests.DefaultULSizes, false, false)

	result := SpeedtestResults{
		Server:        client.Server,
		DownloadSpeed: 1000 * tester.Download(client.Server),
		UploadSpeed:   1000 * tester.Upload(client.Server),
		Latency:       client.Server.Latency / 1000,
	}
	log.Infoln("Speedtest results:", result)
	return result
}
