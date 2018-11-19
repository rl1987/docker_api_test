package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type APIClient struct {
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

func (ac *APIClient) CreateContainer(image string) (string, error) {
	var containerID = ""
	var completeURL = ac.httpServerURL() + "/containers/create"

	containerSpec := map[string]interface{}{
		"Image":        image,
		"AttachStdin":  true,
		"AttachStdout": true,
		"Tty":          true,
		"Entrypoint":   "/bin/bash",
	}

	rqPayloadJSON, err := json.Marshal(containerSpec)
	if err != nil {
		return "", err
	}

	resp, err := ac.httpClient.Post(completeURL, contentType,
		bytes.NewReader(rqPayloadJSON))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	st := struct {
		ContainerId string `json:"Id"`
		Warnings    []string
	}{}

	err = json.NewDecoder(resp.Body).Decode(&st)
	if err != nil {
		return "", err
	}

	containerID = st.ContainerId

	if len(st.Warnings) > 0 {
		for _, w := range st.Warnings {
			fmt.Println("Warning: " + w)
		}
	}

	return containerID, nil
}

func (ac *APIClient) StartContainer(containerID string) error {
	var completeURL = ac.httpServerURL() + "/containers/" + containerID + "/start"

	resp, err := ac.httpClient.Post(completeURL, contentType, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode == 204 {
		return nil
	}

	defer resp.Body.Close()

	return parseErrorMessage(resp)
}

func (ac *APIClient) CheckIfContainerIsRunning(containerID string) (bool, error) {
	var completeURL = ac.httpServerURL() + "/containers/" + containerID + "/json?size=false"

	resp, err := ac.httpClient.Get(completeURL)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		st := struct {
			State struct{ Running bool }
		}{}

		if err = json.NewDecoder(resp.Body).Decode(&st); err != nil {
			return false, err
		}

		return st.State.Running, nil
	}

	return false, parseErrorMessage(resp)
}

func (ac *APIClient) StopContainer(containerID string) error {
	var completeURL = ac.httpServerURL() + "/containers/" + containerID + "/stop"

	resp, err := ac.httpClient.Post(completeURL, contentType, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode == 204 {
		return nil
	}

	defer resp.Body.Close()

	return parseErrorMessage(resp)
}

func (ac *APIClient) RemoveContainer(containerID string) error {
	var completeURL = ac.httpServerURL() + "/containers/" + containerID

	request, _ := http.NewRequest("DELETE", completeURL, nil) // XXX: consider checking error

	resp, err := ac.httpClient.Do(request)
	if err != nil {
		return nil
	}

	if resp.StatusCode == 204 {
		return nil
	}

	defer resp.Body.Close()

	return parseErrorMessage(resp)
}

func (ac *APIClient) CreateExec(containerID string, command []string) (string, error) {
	var completeURL = ac.httpServerURL() + "/containers/" + containerID + "/exec"

	taskSpec := map[string]interface{}{
		"AttachStdout": true,
		"Cmd":          command,
	}

	rqPayloadJSON, err := json.Marshal(&taskSpec)
	if err != nil {
		return "", err
	}

	resp, err := ac.httpClient.Post(completeURL, contentType, bytes.NewReader(rqPayloadJSON))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		st := struct {
			Identifier string `json:"Id"`
		}{}

		err = json.NewDecoder(resp.Body).Decode(&st)
		if err != nil {
			return "", err
		}

		return st.Identifier, nil
	}

	return "", parseErrorMessage(resp)
}

func (ac *APIClient) StartExec(execID string) (string, error) {
	var completeURL = ac.httpServerURL() + "/exec/" + execID + "/start"

	taskSpec := map[string]interface{}{
		"Tty": true,
	}

	rqPayloadJSON, err := json.Marshal(&taskSpec)
	if err != nil {
		return "", err
	}

	resp, err := ac.httpClient.Post(completeURL, contentType, bytes.NewReader(rqPayloadJSON))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		stdout, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(stdout), nil
	}

	return "", parseErrorMessage(resp)
}

func parseErrorMessage(resp *http.Response) error {

	errMsg := struct {
		Message string `json:"message"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&errMsg); err != nil {
		return err
	}

	return errors.New(errMsg.Message)
}
