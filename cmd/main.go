package main

import (
	"flag"
	"fmt"
	"github.com/anishj0shi/load-generator/pkg/http"
	"github.com/anishj0shi/load-generator/pkg/payloads"
	"github.com/anishj0shi/load-generator/pkg/utils"
	"github.com/google/uuid"
	"github.com/prometheus/common/log"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"os"
	"time"
)

func main() {
	token := flag.String("token", "", "Connectivity Token")
	flag.Parse()
	if *token == "" {
		log.Error("Required parameter \"token\" is missing")
		os.Exit(1)
	}

	connectorClient := http.NewConnectorClient(*token)
	gwClient := connectorClient.GetGatewayClient()

	vegetaClient := utils.NewVegetaClient(gwClient.GetHTPClient(), gwClient.GetEventingEndpoint())
	metrics := &vegeta.Metrics{}
	for res := range vegetaClient.Attack(vegeta.Rate{
		Freq: 200,
		Per:  time.Second,
	}, 20*time.Second) {
		metrics.Add(res)
	}
	metrics.Close()

	err := vegeta.NewJSONReporter(metrics).Report(os.Stdout)
	if err != nil {
		panic(err)
	}
	vegeta.NewHDRHistogramPlotReporter(metrics).Report(os.Stdout)

}

func sendEvent(gwClient http.GatewayClient) error {
	event := getEventPayload()
	return gwClient.SendEvent(event)
}

func getEventPayload() payloads.ExampleEvent {
	timeinMillis := time.Now().UTC().UnixNano()

	event := payloads.ExampleEvent{
		EventType:        "bla.event",
		EventTypeVersion: "v1",
		EventID:          uuid.New().String(),
		EventTime:        time.Now().UTC(),
		Data:             fmt.Sprintf("{\"time\" : \"%s\"}", timeinMillis),
	}
	return event
}
