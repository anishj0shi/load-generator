package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/anishj0shi/inmemorydb-service/pkg/schema"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

func main() {
	recordCount := 0
	inMemoryServiceUrl := flag.String("url",
		"http://localhost:8080/eventResult", "URL for InMemoryDB Service")
	flag.Parse()

	f, err := os.OpenFile(fmt.Sprintf("latencylog-%d", time.Now().Unix()),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatalf("error opening file: %v", err)
	}
	tw := tabwriter.NewWriter(f, 0, 8, 2, ' ', tabwriter.StripEscape)
	_, err = fmt.Fprintf(tw, "ID\t\t\tE2ELatency(ms)\n\n")
	if err != nil {
		logrus.Fatalf("error printing headers: %v", err)
	}
	if err := tw.Flush(); err != nil {
		logrus.Fatal(err)
	}

	factor := 10
	top := factor
	skip := 0
	client := &http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       40 * time.Second,
	}
	for {
		var response []schema.EventResult
		url := fmt.Sprintf(*inMemoryServiceUrl+"?top=%d&skip=%d", top, skip)
		logrus.Infof("Calling Service: %s", url)
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			logrus.Warning(err)
			continue
		}
		res, err := client.Do(request)
		if err != nil {
			logrus.Warning(err)
			continue
		}
		logrus.Info(res.Body)
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			logrus.Warningf("Decoding json is a problem %v", err)
		}
		for _, res := range response {
			recordCount++
			err := writeHDRData(f, res, recordCount)
			if err != nil {
				logrus.Warningf("Writing HDR Data is a problem %v", err)
				continue
			}
		}
		skip = top
		top = top + factor

	}
}

func writeHDRData(w io.Writer, res schema.EventResult, count int) error {
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', tabwriter.StripEscape)
	_, err := fmt.Fprintf(tw, "%d\t\t\t%d\n", count, res.E2ELatency)
	if err != nil {
		return err
	}
	err = tw.Flush()
	if err != nil {
		return err
	}
	return nil
}
