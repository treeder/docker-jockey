package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/inconshreveable/log15.v2"
	"io/ioutil"
	"os"
)

type Cluster struct {
	Name      string     `json:"name"`
	Instances []Instance `json:"instances"`
}

func NewCluster(name string) *Cluster {
	c := &Cluster{}
	c.Name = name
	c.Instances = []Instance{}
	return c
}

func LoadCluster(name string) *Cluster {
	c := NewCluster(name)
	file, err := ioutil.ReadFile(c.clusterFile())
	if err != nil {
		log15.Info("No cluster file found, will make new one.")
		return c
	}
	err = json.Unmarshal(file, &c)
	if err != nil {
		log15.Crit("Invalid cluster file!!!", "error:", err)
		os.Exit(1)
	}
	return c
}

func (c *Cluster) clusterFile() string {
	return fmt.Sprintf("jocker.cluster.%v.json", c.Name)
}

func (c *Cluster) AddInstance(i Instance) {
	c.Instances = append(c.Instances, i)
}

func (c *Cluster) GetInstance(name string) *Instance {
	for _, v := range c.Instances {
		if v.Name == name {
			return &v
		}
	}
	return nil
}

func (c *Cluster) HasInstance(name string) bool {
	return c.GetInstance(name) != nil
}

func (c *Cluster) Save() error {
	b, err := json.Marshal(c)
	if err != nil {
		log15.Crit("Error marshalling cluster info", "error", err)
		return err
	}
	err = ioutil.WriteFile(c.clusterFile(), b, 0644)
	if err != nil {
		log15.Crit("Error writing cluster info", "error", err)
		return err
	}
	return nil
}

type Instance struct {
	Name       string `json:"name"` // corresponding to container. This isn't actually right though since one instance might run many containers
	InstanceId string `json:"instance_id"`
}
