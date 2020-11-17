package http

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/anishj0shi/load-generator/pkg/payloads"
	"github.com/anishj0shi/load-generator/pkg/utils"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	log2 "log"
	"net/http"
	"time"
)

type ConnectorClient interface {
	GetGatewayClient() GatewayClient
}

type GatewayClient interface {
	PublishMetadata(details *payloads.ServiceDetails) error
	SendEvent(events payloads.ExampleEvent) error
	GetHTPClient() *http.Client
	GetEventingEndpoint() string
}
type connectorClient struct {
	client         *http.Client
	token          string
	infoResponse   *payloads.InfoResponse
	tlsCertificate []tls.Certificate
}

func NewConnectorClient(token string) ConnectorClient {
	client := &connectorClient{
		client:         getHTTPCLientTemplate(),
		token:          token,
		infoResponse:   nil,
		tlsCertificate: nil,
	}

	client.infoResponse = client.getConnectorResponse()
	client.tlsCertificate = client.getCertificates()

	return client
}

func (c *connectorClient) getConnectorResponse() *payloads.InfoResponse {

	log.Info("Retrieving Token Response")
	request, err := http.NewRequest(http.MethodGet, c.token, nil)
	if err != nil {
		log2.Panicf("Unable to form Request to retrieve Info Response, err: %+v", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		panic(err)
	}
	if response.StatusCode != http.StatusOK {
		log2.Panicf("Response Code REceived: %d", response.StatusCode)
	}
	infoResponse := &payloads.InfoResponse{}

	err = json.NewDecoder(response.Body).Decode(infoResponse)
	if err != nil {
		panic(err)
	}
	return infoResponse
}

func (c *connectorClient) GetGatewayClient() GatewayClient {
	client := getHTTPCLientTemplate()
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.Certificates = c.tlsCertificate
//	transport.DisableKeepAlives = true

	client.Transport = transport

	return newGatewayClient(client, c.infoResponse.Api.EventsUrl, c.infoResponse.Api.MetadataUrl)
}

func (c *connectorClient) getCertificates() []tls.Certificate {
	log.Info("Retrieving Certificates")

	kymaCerts := utils.NewKymaCertificates(c.infoResponse.Certificate.Subject)
	csrRequest := kymaCerts.GetCertificateRequest()

	base64CSRRequest := base64.StdEncoding.EncodeToString(csrRequest)

	body, err := json.Marshal(payloads.CsrRequest{Csr: base64CSRRequest})
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest(http.MethodPost, c.infoResponse.CertUrl, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	request.Header.Add("Content-Type", "application/json")

	response, err := c.client.Do(request)

	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		panic(response)
	}

	crtResponse := &payloads.CrtResponse{}

	err = json.NewDecoder(response.Body).Decode(&crtResponse)
	if err != nil {
		panic(err)
	}
	kymaCerts.AppendCertificates(crtResponse.CRTChain)

	certs, err := kymaCerts.GetCertificates()
	if err != nil {
		panic(err)
	}

	rawChain := make([][]byte, 0, len(certs))
	for _, cert := range certs {
		rawChain = append(rawChain, cert.Raw)
	}
	tlsCert := tls.Certificate{Certificate: rawChain, PrivateKey: kymaCerts.GetCertificateKey()}

	log.Infof("Retrieved Certificated : %+v", tlsCert)
	return []tls.Certificate{tlsCert}
}

type gatewayClient struct {
	client      *http.Client
	eventsUrl   string
	metadataUrl string
}

func newGatewayClient(client *http.Client, eventsUrl, metadataUrl string) GatewayClient {
	return &gatewayClient{
		client:      client,
		eventsUrl:   eventsUrl,
		metadataUrl: metadataUrl,
	}
}

func (g *gatewayClient) GetEventingEndpoint() string  {
	return g.eventsUrl
}

func (g *gatewayClient) GetHTPClient() *http.Client {
	return g.client
}

func (g *gatewayClient) PublishMetadata(details *payloads.ServiceDetails) error {
	return nil
}

func (g *gatewayClient) SendEvent(events payloads.ExampleEvent) error {
	body, err := json.Marshal(events)

	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, g.eventsUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	response, err := g.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("send event failed: %v\nrequest: %v\nresponse: %v", response.StatusCode, request, response)
	}

	return nil
}

func getHTTPCLientTemplate() *http.Client {
	return &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}
