package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/iron-io/golog" // todo: get rid of this usage
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/ec2"
	"github.com/voxelbrain/goptions"
	"gopkg.in/inconshreveable/log15.v2"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	sshttp_port = 8022
)

type Options struct {
	AwsAccessKey string        `goptions:"--aws-access-key, description='AWS Access Key'" json:"aws_access_key"`
	AwsSecretKey string        `goptions:"--aws-secret-key, description='AWS Secret Key'" json:"aws_secret_key"`
	AwsSubnetId  string        `goptions:"--aws-subnet-id, description='AWS Subnet ID to select VPC'" json:"aws_subnet_id"`
	AwsKeyPair   string        `goptions:"--aws-keypair, description='AWS keypair name'" json:"aws_key_pair"`
	SshttpToken  string        `goptions:"--sshttp-token, description='Token for sshttp'" json:"sshttp_token"`
	Help         goptions.Help `goptions:"-h, --help, description='Show this help'"`

	Verb goptions.Verbs
	Run  struct {
		Name        string `goptions:"--name, mutexgroup='input', obligatory, description='Container name'"`
		D           bool   `goptions:"-d"`
		Rm          bool   `goptions:"--rm"`
		Interactive bool   `goptions:"-i, --interactive, description='Force removal'"`
		Tty         bool   `goptions:"-t, --tty, description='Force removal'"`
		Volume      string `goptions:"-v, --volume, obligatory, description='Host dir : container dir'"`
		Workdir     string `goptions:"-w, --workdir, obligatory, description='work dir inside container'"`
		Remainder   goptions.Remainder
	} `goptions:"run"`
	Delete struct {
		Path  string `goptions:"-n, --name, obligatory, description='Name of the entity to be deleted'"`
		Force bool   `goptions:"-f, --force, description='Force removal'"`
	} `goptions:"delete"`
}

func main() {

	options := Options{ // Default values go here
		SshttpToken: "hello",
		AwsKeyPair:  "mykeypair1",
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

	// load existing cluster info if it exists
	cluster := LoadCluster("default")

	// join up the remainder and pass into userData string
	commandString := strings.Join(remainder, " ")
	golog.Infoln("Commandstring:", commandString)

	// Package up the entire directory (could parse -v to see which directory?)
	//	filedir := filepath.Dir(command) // todo: in ruby or interpreted languages, this wouldn't be the file

	//	maybe just use current working dir instead of trying to figure it out, just make that a documented limitation
	//	or the -v would probably be best?  Must include -v
	//
	vsplit := strings.Split(options.Run.Volume, ":")
	log15.Info("dirs", "v0", vsplit[0], "v1", vsplit[1])

	log15.Info("env", "env", os.Getenv("PATH"))

	// Save the last directory

	// zip it up
	tarfile := "script.tar.gz"
	out, err := exec.Command("tar", strings.Fields(fmt.Sprintf("-czf %v %v", tarfile, vsplit[0]))...).CombinedOutput()
	if err != nil {
		log15.Crit("Error tarring", "error", err)
		os.Exit(1)
	}
	log15.Info("Tar ran", "output", string(out))

	e, err := GetEc2(options)
	if err != nil {
		return
	}

	// Check if server already exists/running
	var instance ec2.Instance
	if cluster.HasInstance(options.Run.Name) {
		ins := cluster.GetInstance(options.Run.Name)
		instance, err = GetInstanceInfo(e, ins.InstanceId)
		log15.Info("Instance already exists", "id", instance)
	} else {
		log15.Info("Launching new instance...")
		instance, err := LaunchServer(e, options, options.Run.Name)
		if err != nil {
			log15.Crit("Instance failed to launch", "error", err)
			os.Exit(1)
		}
		cluster.AddInstance(Instance{Name: options.Run.Name, InstanceId: instance.InstanceId})
		err = cluster.Save()
		if err != nil {
			os.Exit(1)
		}
	}
	//	if true {
	//		os.Exit(1)
	//	}

	// Now we're running so let's upload script and run it! upload via sshttp
	// http://stackoverflow.com/questions/20205796/golang-post-data-using-the-content-type-multipart-form-data

	log15.Info("Uploading script...")
	v := url.Values{}
	v.Set("token", options.SshttpToken)
	v.Set("path", "/jocker")
	err = Upload(sshttpUrl(instance, "/v1/files", v), tarfile)
	if err != nil {
		log15.Crit("Error uploading script!", "error", err)
		os.Exit(1)
	}

	// untar
	v = url.Values{}
	log15.Info("Unpacking script...")
	v.Set("token", options.SshttpToken)
	v.Set("exec", "cd /jocker && mkdir script && tar -xf script.tar.gz --strip 1 --directory script")
	resp, err := http.Post(sshttpUrl(instance, "/v1/shell", v), "application/json", strings.NewReader("{}"))
	if err != nil {
		log15.Crit("Couldn't untar script", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // I think there's a better way to do this so it doesn't read it into memory, io.Copy into nil or something
	log15.Info("Untar run", "output", string(body))

	// run the docker command! could setup an upstart script for it too if it's a service (optional)
	log15.Info("Running script...")
	v = url.Values{}
	v.Set("token", options.SshttpToken)
	v.Set("exec", fmt.Sprintf("cd /jocker/script && docker run -it --rm --name yo -v \"$(pwd)\":/usr/src/myapp -w /usr/src/myapp %v", commandString))
	resp, err = http.Post(sshttpUrl(instance, "/v1/shell", v), "application/json", strings.NewReader("{}"))
	if err != nil {
		log15.Crit("Couldn't run script in docker!", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body) // I think there's a better way to do this so it doesn't read it into memory, io.Copy into nil or something
	log15.Info("Docker run", "output", string(body))

}

func GetEc2(options Options) (*ec2.EC2, error) {
	auth, err := aws.GetAuth(options.AwsAccessKey, options.AwsSecretKey)
	if err != nil {
		log15.Crit("Error aws.GetAuth", "error", err)
		return nil, err
	}
	e := ec2.New(auth, aws.USEast)
	return e, nil
}

func GetInstanceInfo(e *ec2.EC2, instanceId string) (instance ec2.Instance, err error) {
	iResp, err := e.Instances([]string{instanceId}, nil)
	if err != nil {
		log15.Crit("Couldn't get instance details", "error", err)
		return instance, err
	}
	instance = iResp.Reservations[0].Instances[0]
	return instance, err
}

func LaunchServer(e *ec2.EC2, options Options, name string) (instance ec2.Instance, err error) {

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
mkdir /jocker
chmod 777 /jocker
curl -L -O https://github.com/treeder/sshttp/releases/download/v0.0.1/sshttp
chmod +x sshttp
./sshttp -t hello
`
	// todo: MAKE SSHTTP TOKEN CONFIGURABLE!!!  MAKE RANDOM TOKEN AT FIRST AND STORE THAT ALONG WITH CLUSTER CONFIG

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
		return instance, err
	}

	for _, instance := range resp.Instances {
		log15.Info("Now running", "instance", instance.InstanceId)
	}
	log15.Info("Make sure you terminate instances to stop the cash flow!")

	// Now we'll wait until it fires up and sshttp is available

	ok := false
	ticker := time.NewTicker(time.Second * 2)
L:
	for {
		select {
		case <-ticker.C:
			instance, err = GetInstanceInfo(e, resp.Instances[0].InstanceId)
			if err != nil {
				log15.Crit("Error getting instance info", "error", err)
				os.Exit(1)
			}
			if checkIfUp(instance) {
				ok = true
				break L
			}
		case <-time.After(5 * time.Minute):
			ticker.Stop()
			log15.Warn("Timed out trying to connect.")
			break
		}
	}
	if !ok {
		return instance, err
	}
	log15.Info("Instance is running and sshttp is online.")
	_, err = e.CreateTags([]string{instance.InstanceId}, []ec2.Tag{ec2.Tag{Key: "Name", Value: options.Run.Name}})
	if err != nil {
		log15.Crit("Error creating tags!", "error", err)
		return instance, err
	}
	return instance, err
}

func sshttpUrlBase(i ec2.Instance) string {
	return fmt.Sprintf("http://%v:%v", i.DNSName, sshttp_port)
}

func sshttpUrl(i ec2.Instance, path string, v url.Values) string {
	return fmt.Sprintf("%v%v?%v", sshttpUrlBase(i), path, v.Encode())
}

func checkIfUp(i ec2.Instance) bool {
	log15.Info("Checking instance status", "state", i.State, "id", i.InstanceId)
	// check if sshttp available
	if i.DNSName != "" { // wait for it to get a public dns entry (takes a bit)
		resp, err := http.Get(sshttpUrlBase(i))
		if err != nil {
			log15.Info("sshttp not available yet", "error", err)
			return false
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log15.Warn("Couldn't read sshttp body", "error", err)
			return false
		}
		log15.Info("response", "body", string(body))
		return true
	}
	return false
}

func Upload(url, file string) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return
	}
	fw, err := w.CreateFormFile("file", file)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return err
}
