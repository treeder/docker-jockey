package main

import (
	"encoding/json"
	"github.com/iron-io/golog" // todo: get rid of this usage
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/ec2"
	"github.com/voxelbrain/goptions"
	"gopkg.in/inconshreveable/log15.v2"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"net/http"
	"fmt"
)

var (
	sshttp_port = 8022
)

func main() {

	options := struct {
		AwsAccessKey string        `goptions:"--aws-access-key, description='AWS Access Key'" json:"aws_access_key"`
		AwsSecretKey string        `goptions:"--aws-secret-key, description='AWS Secret Key'" json:"aws_secret_key"`
		AwsSubnetId  string        `goptions:"--aws-subnet-id, description='AWS Subnet ID to select VPC'" json:"aws_subnet_id"`
		AwsKeyPair   string        `goptions:"--aws-keypair, description='AWS keypair name'" json:"aws_key_pair"`
		Help         goptions.Help `goptions:"-h, --help, description='Show this help'"`

		Verb goptions.Verbs
		Run  struct {
			Name        string `goptions:"--name, mutexgroup='input', description='Container name'"`
			Rm          bool   `goptions:"--rm"`
			Interactive bool   `goptions:"-i, --interactive, description='Force removal'"`
			Tty    bool   `goptions:"-t, --tty, description='Force removal'"`
			Volume      string `goptions:"-v, --volume, obligatory, description='Name of the entity to be deleted'"`
			Workdir     string `goptions:"-w, --workdir, obligatory, description='Name of the entity to be deleted'"`
			Remainder   goptions.Remainder
		} `goptions:"run"`
		Delete struct {
			Path  string `goptions:"-n, --name, obligatory, description='Name of the entity to be deleted'"`
			Force bool   `goptions:"-f, --force, description='Force removal'"`
		} `goptions:"delete"`
	}{ // Default values go here
		AwsKeyPair: "mykeypair1",
	}

	// load from file first if it exists
	// todo; should build this into goptions or make a new config lib that uses this + goptions
	file, err := ioutil.ReadFile("jocker.config.json")
	if err != nil {
		log15.Info("No config file found.")
	} else {
		err = json.Unmarshal(file, &options)
		if err != nil {
			log15.Crit("Invalid config file!!!", "error:", err)
			os.Exit(1)
		}
//		log15.Debug("Results:", "jsoned", options)
	}

	goptions.ParseAndFail(&options)

	golog.Infoln("REMAINDER:", options.Run.Remainder)
	// parse remainder to get image and command. sudo docker run [OPTIONS] IMAGE[:TAG] [COMMAND] [ARG...]
	remainder := options.Run.Remainder
	image := remainder[0]
	command := remainder[1]
	commandArgs := remainder[2:]
	golog.Infoln("XXX", image, command, commandArgs)

	// join up the remainder and pass into userData string
	commandString := strings.Join(remainder, " ")
	golog.Infoln("Commandstring:", commandString)

	// Package up the entire directory (could parse -v to see which directory?)

	if false {
		panic("YO!")
	}

	auth, err := aws.GetAuth(options.AwsAccessKey, options.AwsSecretKey)
	if err != nil {
		golog.Errorln(err)
		return
	}
	e := ec2.New(auth, aws.USEast)

	//	userData := []byte(`#cloud-config
	//runcmd:
	//- [ wget, "http://slashdot.org", -O, /tmp/index.html ]
	//- [ sh, -xc, "echo $(date) ': hello world!'" ]
	//- [ curl, -sSL, "https://get.docker.io/ubuntu/" | sudo sh ]
	//`)
	userDataString := `#!/bin/sh
echo "Docker Jockey is spinning! And the time is now $(date -R)!"
echo "Current dir: $(pwd)"
curl -sSL https://get.docker.io/ubuntu/ | sudo sh
curl -L -O https://github.com/treeder/sshttp/releases/download/v0.0.1/sshttp
chmod +x sshttp
./sshttp
`

	// Now add the docker command to it too to run the users script

	userData := []byte(userDataString)
	// todo: install sshttp on the machine during init too

	ec2Options := ec2.RunInstances{
		ImageId:      "ami-10389d78", // 14.04, EBS store
		InstanceType: "t2.small",
		SubnetId:     options.AwsSubnetId,
		KeyName:      options.AwsKeyPair,
		UserData:     userData,
	}
	resp, err := e.RunInstances(&ec2Options)
	if err != nil {
		golog.Errorln(err)
		return
	}

	for _, instance := range resp.Instances {
		log15.Info("Now running", "instance", instance.InstanceId)
	}
	log15.Info("Make sure you terminate instances to stop the cash flow!")

	// Now we'll wait until it fires up and sshttp is available
	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			iResp, err := e.Instances([]string{resp.Instances[0].InstanceId}, nil)
			if err != nil {
				log15.Crit("Couldn't get instance details", "error", err)
				os.Exit(1)
			}
			checkIfUp(iResp.Reservations[0].Instances[0])
		case <-time.After(5 * time.Minute):
			ticker.Stop()
			log15.Warn("Timed out trying to connect.")
		}
	}


}

func checkIfUp(i ec2.Instance) {

	log15.Info("Checking instance status", "state", i.State, "id", i.InstanceId)
	// check if sshttp available
	if i.DNSName != "" { // wait for it to get a public dns entry (takes a bit)
		resp, err := http.Get(fmt.Sprintf("http://%v:%v", i.DNSName, sshttp_port))
		if err != nil {
			log15.Info("sshttp not available yet", "error", err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		log15.Info("response", "body", body)
	}
}
