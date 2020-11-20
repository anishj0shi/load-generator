package main

import (
	"flag"
	"fmt"
	"github.com/anishj0shi/load-generator/pkg/http"
	"github.com/anishj0shi/load-generator/pkg/payloads"
	"github.com/anishj0shi/load-generator/pkg/utils"
	"github.com/google/uuid"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"os"
	"time"
)

func main() {
	token := flag.String("token", "", "Connectivity Token")
	//frequency := flag.Int("frequency", "80", "Number of requests per second")
	flag.Parse()
	if *token == "" {
		log.Error("Required parameter \"token\" is missing")
		os.Exit(1)
	}

	connectorClient := http.NewConnectorClient(*token)
	gwClient := connectorClient.GetGatewayClient()

	vegetaClient := utils.NewVegetaClient(gwClient.GetHTPClient(), gwClient.GetEventingEndpoint())
	metrics := &vegeta.Metrics{}
	pacer := vegeta.SinePacer{
		Period: 2 * time.Minute,
		Mean: vegeta.Rate{
			Freq: 20,
			Per:  time.Second,
		},
		Amp: vegeta.Rate{
			Freq: 5,
			Per:  time.Second,
		},
		StartAt: 0,
	}
	//vegeta.Rate{
	//	Freq: 20,
	//	Per:  time.Second,
	//}
	for res := range vegetaClient.Attack(pacer, 60*time.Second) {
		metrics.Add(res)
	}
	metrics.Close()

	f1, err := os.OpenFile(fmt.Sprintf("jsonlog-%d", time.Now().Unix()),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatalf("error opening file: %v", err)
	}

	f2, err := os.OpenFile(fmt.Sprintf("hdrlog-%d", time.Now().Unix()),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatalf("error opening file: %v", err)
	}

	err = vegeta.NewJSONReporter(metrics).Report(f1)
	if err != nil {
		panic(err)
	}

	err = vegeta.NewHDRHistogramPlotReporter(metrics).Report(f2)
	if err != nil {
		panic(err)
	}

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
