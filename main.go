package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

const imageName = "ubuntu"
const imageTag = "latest"

const imageToPull = imageName + ":" + imageTag

const contentType = "application/json"

type DockerImage struct {
	Identifier  string   `json:"Id"`
	RepoDigests []string `json:"RepoDigests"`
	RepoTags    []string `json:"RepoTags"`
}

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

func (ac *APIClient) FindImage(image string) ([]DockerImage, error) {
	filter := map[string][]string{
		"reference": []string{image},
	}

	filtersJson, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	var result []DockerImage
	url := "/images/json?filters=" + string(filtersJson)

	if err = ac.Get(url, &result); err != nil {
		spew.Dump(err)
		return nil, err
	}

	if *debug {
		spew.Dump(result)
	}

	return result, nil
}

func (ac *APIClient) PullImage(image string) (string, error) {
	var digest = ""
	var completeURL = ac.httpServerURL() + "/images/create?fromImage=" + image

	resp, err := ac.httpClient.Post(completeURL, contentType, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	st := struct {
		Status string `json:"status"`
	}{}

	var unmarshalErr error

	decoder := json.NewDecoder(resp.Body)

	for {
		if unmarshalErr = decoder.Decode(&st); unmarshalErr == nil {
			fmt.Println(st.Status)

			if strings.HasPrefix(st.Status, "Digest: sha256:") {
				digest = strings.TrimPrefix(st.Status, "Digest: sha256:")
			}
		} else {
			if unmarshalErr != io.EOF {
				spew.Dump(unmarshalErr)
				return digest, unmarshalErr
			}

			break
		}
	}

	return digest, nil
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

	images, err := apiClient.FindImage(imageToPull)
	if err != nil {
		fmt.Println(err)
	}

	var imageDigest string

	if len(images) == 0 {
		imageDigest, err = apiClient.PullImage(imageToPull)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		imageDigest = strings.TrimPrefix(images[0].Identifier, "sha256:")
	}

	fmt.Println(imageDigest)
}
