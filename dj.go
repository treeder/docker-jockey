package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Version     = "0.0.5"
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
		// todo: should allow an instance name to allow multiple containers on one instance (default is name already)
		On          string `goptions:"--on, description='Which cloud to run this on. Omit if want to run locally.'"`
		Name        string `goptions:"--name, description='Container name'"`
		D           bool   `goptions:"-d"`
		Port        string `goptions:"-p, description='Port setup, eg: 0.0.0.0:8080:8080'"`
		Rm          bool   `goptions:"--rm"`
		Interactive bool   `goptions:"-i, --interactive, description='Force removal'"`
		Tty         bool   `goptions:"-t, --tty, description='Force removal'"`
		Volume      string `goptions:"-v, --volume, description='Host dir : container dir'"`
		Workdir     string `goptions:"-w, --workdir, description='work dir inside container'"`
		Remainder   goptions.Remainder
	} `goptions:"run"`
	Stop struct {
		Time      int `goptions:"-t, --time, description='Number of seconds to wait for the container to stop before killing it. Default is 10 seconds.'"`
		Remainder goptions.Remainder
	} `goptions:"stop"`
	Ssh struct {
		Name      string `goptions:"--name, description='Container name'", obligatory`
		Remainder goptions.Remainder
	} `goptions:"ssh"`
	Version struct {
		Remainder goptions.Remainder
	} `goptions:"version"`
}

func main() {

	// Default values go here
	options := &Options{
		SshttpToken: "hello",
		AwsKeyPair:  "mykeypair1",
	}
	options.Run.Port = "8080:8080" // default value
	options.Run.Rm = true
	wd, err := os.Getwd()
	if err != nil {
		log15.Crit("Couldn't get currentw working dir", "error", err)
		os.Exit(1)
	}
	options.Run.Volume = wd + ":/app"
	options.Run.Workdir = "/app"

	// load from file first if it exists
	// todo; should build this into goptions or make a new config lib that uses this + goptions
	file, err := ioutil.ReadFile("dj.config.json")
	if err != nil {
		log15.Info("No config file found.")
	} else {
		err = json.Unmarshal(file, options)
		if err != nil {
			log15.Crit("Invalid config file!!!", "error:", err)
			os.Exit(1)
		}
		//		log15.Debug("Results:", "jsoned", options)
	}
	goptions.ParseAndFail(options)

	// load existing cluster info if it exists
	cluster := LoadCluster("default")

	switch options.Verb {
	case "run":
		Run(options, cluster)
	case "stop":
		Stop(options, cluster)
	case "ssh":
		Ssh(options, cluster)
	case "version":
		fmt.Println(Version)
	}

}

func Ssh(options *Options, cluster *Cluster) {
	ec2i, err := GetEc2(options)
	if err != nil {
		return
	}
	ins := cluster.GetInstance(options.Ssh.Name)
	instance, err := GetInstanceInfo(ec2i, ins.InstanceId)
	if err != nil {
		log15.Warn("Instance not on AWS anymore.", "id", ins.InstanceId)
		cluster.RemoveInstance(options.Ssh.Name)
	} else {
		if instance.State.Code == 16 { // running code: http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-InstanceStateType.html
			// instOk = true
			log15.Info("Instance already exists, using it.", "id", instance)
		} else {
			log15.Warn("Instance no longer in running state")
		}
	}
	cmd := strings.Join(options.Ssh.Remainder, " ")
	output, err := remoteExec(options, instance.DNSName, cmd)
	if err != nil {
		log15.Error("Remote command failed!", "error", err, "output", output, "instance_id", instance.InstanceId, "host", instance.DNSName)
	}
	log15.Info("Remote cmd successful.", "output", output)
}

func Run(options *Options, cluster *Cluster) {

	if options.Run.On == "aws" {
		Deploy(options, cluster)
	} else {
		RunLocal(options, cluster)
	}

}

func RunLocal(options *Options, cluster *Cluster) {
	fields := []string{"run"}
	fields = append(fields, "-v", options.Run.Volume, "-w", options.Run.Workdir, "-p", options.Run.Port)
	fields = append(fields, "--rm", "-i")
	if options.Run.Tty {
		fields = append(fields, "-t")
	}
	if options.Run.Name != "" {
		fields = append(fields, "--name", options.Run.Name)
	}
	fields = append(fields, options.Run.Remainder...)
	fmt.Println(fields)
	cmd := exec.Command("docker", fields...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//	out, err := exec.Command("pwd").CombinedOutput()
	err := cmd.Run()
	if err != nil {
		log15.Error("Error occured", "error", err)
	}
}

func Deploy(options *Options, cluster *Cluster) {
	log15.Info("REMAINDER:", "remainder", options.Run.Remainder)
	// parse remainder to get image and command. sudo docker run [OPTIONS] IMAGE[:TAG] [COMMAND] [ARG...]
	remainder := options.Run.Remainder
	log15.Info("remainder", "r", remainder)
	//	image := remainder[0]
	//	command := remainder[1]
	//	commandArgs := remainder[2:]

	// join up the remainder and pass into userData string
	commandString := strings.Join(remainder, " ")
	log15.Info("Command", "string", commandString)

	// Package up the entire directory (could parse -v to see which directory?)
	//	filedir := filepath.Dir(command) // todo: in ruby or interpreted languages, this wouldn't be the file

	//	maybe just use current working dir instead of trying to figure it out, just make that a documented limitation
	//	or the -v would probably be best?  Must include -v
	//
	log15.Info("env", "env", os.Getenv("PATH"))

	zipfile := ""
	if len(remainder) > 1 {
		if options.Run.Volume != "" {
			// Then tar/upload code, otherwise nothing to run, just image
			vsplit := strings.Split(options.Run.Volume, ":")
			log15.Info("dirs", "v0", vsplit[0], "v1", vsplit[1])

			// Save the last directory

			// zip it up
			zipfile = "app.zip"
			err := os.Remove(zipfile)
			if err != nil {
				log15.Info("deleting zip", "err", err)
			}

			// below doesn't work with directories with spaces
			//	fields := strings.Fields(fmt.Sprintf("-czf %v %v", tarfile, vsplit[0]))
			fields := []string{}
			// was using this for the tar file: vsplit[0], but it was messing up
			// fields = append(fields, "-czf", tarfile, ".", "--exclude='" + tarfile + "'") // dot added because of this: http://stackoverflow.com/a/18681628/105562
			// log15.Info("fields", "fields", fields)
			// out, err := exec.Command("tar", fields...).CombinedOutput()
			fields = append(fields, "-r", zipfile, ".") // dot added because of this: http://stackoverflow.com/a/18681628/105562
			log15.Info("fields", "fields", fields)
			out, err := exec.Command("zip", fields...).CombinedOutput()
			if err != nil {
				log15.Crit("Error zipping", "error", err, "out", string(out))
				os.Exit(1)
			}
			log15.Info("Zip ran", "output", string(out))
		}
	}

	ec2i, err := GetEc2(options)
	if err != nil {
		return
	}

	// Check if server already exists/running
	var instance ec2.Instance
	instOk := false
	if cluster.HasInstance(options.Run.Name) {
		ins := cluster.GetInstance(options.Run.Name)
		instance, err = GetInstanceInfo(ec2i, ins.InstanceId)
		if err != nil {
			log15.Warn("Instance not on AWS anymore.", "id", ins.InstanceId)
			cluster.RemoveInstance(options.Run.Name)
		} else {
			if instance.State.Code == 16 { // running code: http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-ItemType-InstanceStateType.html
				instOk = true
				log15.Info("Instance already exists, using it.", "id", instance)
			} else {
				log15.Warn("Instance no longer in running state")
			}
		}
	}
	if !instOk {
		log15.Info("Launching new instance...")
		instance, err = LaunchServer(ec2i, options, options.Run.Name)
		if err != nil {
			log15.Crit("Instance failed to launch", "error", err)
			os.Exit(1)
		}
		cluster.AddInstance(Instance{
			Name:       options.Run.Name,
			InstanceId: instance.InstanceId,
		})

	}
	err = cluster.Save()
	if err != nil {
		os.Exit(1)
	}
	//	if true {
	//		os.Exit(1)
	//	}

	// Now we're running so let's upload script and run it! upload via sshttp
	// http://stackoverflow.com/questions/20205796/golang-post-data-using-the-content-type-multipart-form-data

	if zipfile != "" {
		log15.Info("Uploading script...")
		v := url.Values{}
		v.Set("token", options.SshttpToken)
		v.Set("path", "/dj")
		err = Upload(sshttpUrl(instance.DNSName, "/v1/files", v), zipfile)
		if err != nil {
			log15.Crit("Error uploading script!", "error", err)
			os.Exit(1)
		}

		// untar
		cmd := "cd /dj && rm -rf app && mkdir app && unzip app.zip -d app/"
		output, err := remoteExec(options, instance.DNSName, cmd)
		if err != nil {
			log15.Error("Remote command failed!", "error", err, "output", output, "instance_id", instance.InstanceId, "host", instance.DNSName)
		}
		log15.Info("Untar run", "output", output)
	}

	// run the docker command! could setup an upstart script for it too if it's a service (optional)
	log15.Info("Running script...")
	// todo: If server was already running, we could maybe just do docker stop then docker start instead of rm then run.

	// todo: drop Sprintf's where not needed
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("docker stop %v ; docker rm %v ;", options.Run.Name, options.Run.Name))
	if options.Run.Volume != "" {
		// if no volume, then the image is probably already good to go
		// TODO: should maybe check command instead
		buffer.WriteString(fmt.Sprintf(" cd /dj/app && "))
	}
	buffer.WriteString(fmt.Sprintf(" docker run -d --name %v", options.Run.Name))
	if options.Run.Volume != "" {
		// todo: change to my app
		buffer.WriteString(fmt.Sprintf(" -v /dj/app:/app -w /app"))
	}
	buffer.WriteString(fmt.Sprintf(" -p %v %v",
		options.Run.Port, commandString))

	cmd := buffer.String()
	output, err := remoteExec(options, instance.DNSName, cmd)
	if err != nil {
		log15.Error("Docker run failed!", "error", err, "output", output, "instance_id", instance.InstanceId, "host", instance.DNSName)
	}
	log15.Info("Docker run", "output", output, "instance_id", instance.InstanceId, "host", instance.DNSName)
}

func Stop(options *Options, cluster *Cluster) {
	log15.Info("REMAINDER:", "remainder", options.Stop.Remainder)
	// parse remainder to get image and command. sudo docker run [OPTIONS] IMAGE[:TAG] [COMMAND] [ARG...]
	remainder := options.Stop.Remainder
	if len(remainder) < 1 {
		log15.Error("You need to specify a container.")
		return
	}
	container := remainder[0]

	imeta := cluster.GetInstance(container)
	if imeta == nil {
		log15.Info("Instance does not exist", "instance", container, "cluster", cluster)
		return
	}

	cmd := fmt.Sprintf("docker stop %v", container)
	output, err := remoteExec(options, imeta.Host, cmd)
	if err != nil {
		// todo: ???
	}
	log15.Info("docker stop", "output", output)

	// todo: if there is another container running on this machine, don't terminate
	// now terminate
	ec2, err := GetEc2(options)
	if err != nil {
		return
	}
	resp, err := ec2.TerminateInstances([]string{imeta.InstanceId})
	if err != nil {
		log15.Error("Terminate instances", "error", err)
	}
	log15.Info("Terminate instances", "response", resp)
}

func remoteExec(options *Options, host, cmd string) (string, error) {
	log15.Info("Remote exec", "cmd", cmd)
	v := url.Values{}
	v.Set("token", options.SshttpToken)
	// use semi-colon for first two in case it doesn't exist which will return an error
	v.Set("exec", cmd)
	resp, err := http.Post(sshttpUrl(host, "/v1/shell", v), "application/json", strings.NewReader("{}"))
	if err != nil {
		log15.Crit("Error executing remote command!", "error", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // I think there's a better way to do this so it doesn't read it into memory, io.Copy into nil or something
	return string(body), err
}

func GetEc2(options *Options) (*ec2.EC2, error) {
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
	log15.Debug("GetInstanceInfo", "response", iResp)
	if len(iResp.Reservations) == 0 {
		// instance no longer there
		return instance, fmt.Errorf("Instance not found on aws.")
	}
	instance = iResp.Reservations[0].Instances[0]
	return instance, err
}

func LaunchServer(e *ec2.EC2, options *Options, name string) (instance ec2.Instance, err error) {

	//	userData := []byte(`#cloud-config
	//runcmd:
	//- [ wget, "http://slashdot.org", -O, /tmp/index.html ]
	//- [ sh, -xc, "echo $(date) ': hello world!'" ]
	//- [ curl, -sSL, "https://get.docker.io/ubuntu/" | sudo sh ]
	//`)
	userDataString := `#!/bin/sh
echo "Docker Jockey is spinning! And the time is now $(date -R)!"
echo "Current dir: $(pwd)"
apt-get install unzip
curl -sSL https://get.docker.io/ubuntu/ | sudo sh
mkdir /dj
chmod 777 /dj
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
		log15.Error("Error running instances", "error", err)
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
	_, err = e.CreateTags([]string{instance.InstanceId},
		[]ec2.Tag{
			ec2.Tag{Key: "Name", Value: options.Run.Name},
			ec2.Tag{Key: "dj", Value: "booyah"},
		})
	if err != nil {
		log15.Crit("Error creating tags!", "error", err)
		return instance, err
	}
	return instance, err
}

func sshttpUrlBase(host string) string {
	return fmt.Sprintf("http://%v:%v", host, sshttp_port)
}

func sshttpUrl(host, path string, v url.Values) string {
	return fmt.Sprintf("%v%v?%v", sshttpUrlBase(host), path, v.Encode())
}

func checkIfUp(i ec2.Instance) bool {
	log15.Info("Checking instance status", "state", i.State, "id", i.InstanceId)
	// check if sshttp available
	if i.DNSName != "" { // wait for it to get a public dns entry (takes a bit)
		resp, err := http.Get(sshttpUrlBase(i.DNSName))
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
