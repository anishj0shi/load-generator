package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/anishj0shi/inmemorydb-service/pkg/schema"
	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

var (
	inMemoryServiceUrl string
	retryOpts          = []retry.Option{
		retry.Attempts(5),
		retry.Delay(4 * time.Second),
		retry.DelayType(retry.FixedDelay),
	}
)

func main() {
	flag.StringVar(&inMemoryServiceUrl, "url",
		"http://localhost:8080/eventResult", "URL for InMemoryDB Service")
	flag.Parse()

	collectLog()
}

func collectLog() {
	recordCount := 0
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
		if recordCount > 10000 {
			break
		}
		url := fmt.Sprintf(inMemoryServiceUrl+"?top=%d&skip=%d", top, skip)
		logrus.Infof("Calling Service: %s", url)
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			logrus.Warning(err)
			continue
		}

		err = retry.Do(func() error {
			res, err := client.Do(request)
			if err != nil {
				logrus.Warning(err)
				return err
			}
			var response []schema.EventResult
			logrus.Info(res.Body)
			if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
				logrus.Warningf("Decoding json is a problem %v", err)
			}
			if len(response) == 0 {
				return errors.New(fmt.Sprintf("Empty Response for top = %d, skip = %d", top, skip))
			}
			for _, res := range response {
				recordCount++
				err := writeHDRData(f, res, recordCount)
				if err != nil {
					logrus.Warningf("Writing HDR Data is a problem %v", err)
					continue
				}
			}
			response = nil
			return nil
		}, retryOpts...)
		if err != nil {
			logrus.Warningf("Error : %v", err)
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
