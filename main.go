package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/nsf/termbox-go"
)

const imageName = "ubuntu"
const imageTag = "latest"

const imageToPull = imageName + ":" + imageTag

const contentType = "application/json"

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

	err := termbox.Init()
	checkError(err)

	apiClient, _ := NewAPIClient(*unixAddr, *tcpAddr)
	if *debug {
		spew.Dump(apiClient)
	}

	images, err := apiClient.FindImage(imageToPull)
	checkError(err)

	var imageDigest string

	if len(images) == 0 {
		imageDigest, err = apiClient.PullImage(imageToPull)
		checkError(err)
	} else {
		imageDigest = strings.TrimPrefix(images[0].Identifier, "sha256:")
	}

	fmt.Println("Image: " + imageDigest)

	containerID, err := apiClient.CreateContainer(imageToPull)
	checkError(err)

	fmt.Println("Created container " + containerID)

	err = apiClient.StartContainer(containerID)
	checkError(err)

	fmt.Println("Started container")

	fmt.Printf("Waiting")

	for {
		fmt.Printf(".")
		isRunning, err := apiClient.CheckIfContainerIsRunning(containerID)

		if isRunning {
			fmt.Printf("\n")
			break
		}

		checkError(err)

		time.Sleep(1 * time.Second)
	}

	qPressed := make(chan struct{})

	waitForQFunc := func(qPressed chan struct{}) {
		for {
			if ev := termbox.PollEvent(); ev.Type == termbox.EventKey {
				if ev.Ch == 'q' {
					qPressed <- struct{}{}
					break
				}
			}
		}
	}

	stdoutChan := make(chan string)

	checkLoadFunc := func(containerID string) {
		command := "top -bn1"

		execID, err := apiClient.CreateExec(containerID, strings.Fields(command))
		checkError(err)

		stdoutChan <- "Exec ID: " + execID

		output, err := apiClient.StartExec(execID)
		checkError(err)

		stdoutChan <- output
	}

	go waitForQFunc(qPressed)

	finished := false

	for !finished {
		select {
		case s := <-stdoutChan:
			fmt.Println(s)
			break
		case <-qPressed:
			finished = true
			break
		case <-time.After(1 * time.Second):
			go checkLoadFunc(containerID)
		}

	}

	fmt.Println("Stopping container")
	err = apiClient.StopContainer(containerID)
	checkError(err)

	fmt.Println("Removing container")
	err = apiClient.RemoveContainer(containerID)
	checkError(err)

	termbox.Close()
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		termbox.Close()
		os.Exit(-1)
	}
}
