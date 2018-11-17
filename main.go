package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/davecgh/go-spew/spew"
)

const imageName = "ubuntu"
const imageTag = "latest"

const imageToPull = imageName + ":" + imageTag

const contentType = "application/json"

type APIClient struct {
	HubUsername string
	HubPassword string

	addr       string
	httpClient http.Client
	transport  string
}

func NewAPIClient(unixAddr string, tcpAddr string) (*APIClient, error) {
	var transport string
	var addr string

	if unixAddr != "" {
		transport = "unix"
		addr = unixAddr
	} else if tcpAddr != "" {
		transport = "tcp"
		addr = tcpAddr
	} else {
		return nil, errors.New("Either UNIX socket path or TCP HTTP address has to be provided")
	}

	return &APIClient{
		addr: addr,
		httpClient: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial(transport, addr)
				},
			},
		},
		transport: transport,
	}, nil
}

func (ac *APIClient) httpServerURL() string {
	if ac.transport == "unix" {
		return "http://unix"
	} else if ac.transport == "tcp" {
		return "http://" + ac.addr
	}

	return ""
}

func (ac *APIClient) Get(url string, result interface{}) error {
	var completeURL = ac.httpServerURL() + url

	resp, err := ac.httpClient.Get(completeURL)
	if err != nil {
		return err
	}

	if *debug {
		spew.Dump(resp)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unexpected HTTP respose status: %d",
			resp.StatusCode)
	}

	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

func (ac *APIClient) PullImage(image string) error {
	var completeURL = ac.httpServerURL() + "/images/create?fromImage=" + image

	resp, err := ac.httpClient.Post(completeURL, contentType, nil)

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(payload))

	return nil
}

var unixAddr = flag.String("unixAddr", "", "UNIX socket that provides Docker Engine API")
var tcpAddr = flag.String("tcpAddr", "", "TCP HTTP address for Docker Engine API")
var helpNeeded = flag.Bool("h", false, "usage help")
var debug = flag.Bool("d", false, "print debugging info")

func main() {
	flag.Parse()

	if *helpNeeded || len(os.Args) == 1 {
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return
	}

	apiClient, _ := NewAPIClient(*unixAddr, *tcpAddr)
	if *debug {
		spew.Dump(apiClient)
	}

	/*
		var result interface{}
		if err := apiClient.Get("/v1.36/info", &result); err != nil {
			spew.Dump(err)
			return
		}

		spew.Dump(result)
	*/

	if err := apiClient.PullImage(imageToPull); err != nil {
		fmt.Println(err)
	}
}
