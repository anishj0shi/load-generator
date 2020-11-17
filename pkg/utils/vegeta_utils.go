package utils

import (
	"encoding/json"
	"fmt"
	"github.com/anishj0shi/load-generator/pkg/payloads"
	"github.com/google/uuid"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"net/http"
	"time"
)

type VegetaEventTarget struct {
	url    string
	client *http.Client
}

func NewVegetaClient(client *http.Client, url string) *VegetaEventTarget {
	return &VegetaEventTarget{
		url:    url,
		client: client,
	}
}

func (v *VegetaEventTarget) Attack(rate vegeta.Rate, duration time.Duration) <-chan *vegeta.Result {
	attacker := v.getVegetaAttacker()
	return attacker.Attack(v.getVegetaTarget(), rate, duration, fmt.Sprint("Attack-%d", time.Now().Unix()))
}

func (v *VegetaEventTarget) getVegetaAttacker() *vegeta.Attacker {
	return vegeta.NewAttacker(vegeta.Client(v.client))
}

func (v *VegetaEventTarget) getVegetaTarget() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		t.Method = http.MethodPost
		t.URL = v.url

		event := payloads.ExampleEvent{
			EventType:        "bla.event",
			EventTypeVersion: "v1",
			EventID:          uuid.New().String(),
			EventTime:        time.Now().UTC(),
			Data:             payloads.ExampleEventData{Timestamp: time.Now().Unix()},
		}

		jsonStr, err := json.Marshal(event)
		if err != nil {
			panic(err)
		}
		t.Body = jsonStr

		return nil
	}
}

func getLoadTestingRateFuctions() []vegeta.Rate {
	rates := []vegeta.Rate{{
		Freq: 10,
		Per:  time.Second,
	}, {
		Freq: 30,
		Per:  time.Second,
	}, {
		Freq: 60,
		Per:  time.Second,
	}, {
		Freq: 90,
		Per:  time.Second,
	}, {
		Freq: 120,
		Per:  time.Second,
	}, {
		Freq: 200,
		Per:  time.Second,
	},
	}

	return rates
}
