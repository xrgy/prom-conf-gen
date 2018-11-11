package main

import (
	"github.com/coreos/etcd/client"
	"os"
	"time"
	"log"
	"context"
	"prom-conf-gen/handlers"
)

func checkEtcdConnect(key string,cfg client.Config) error {
	c,err := client.New(cfg)
	if err !=nil{
		log.Printf("create etcd client error: %s",err.Error())
		return err
	}
	kapi := client.NewKeysAPI(c)
	_,err = kapi.Get(context.Background(),key,nil)
	if err!=nil {
		log.Printf("get key %s error: %s",key,err.Error())
		return err
	}
	return nil
}

func main() {
	cfg := client.Config{
		Endpoints: []string{"http://"+os.Getenv("ETCD_ENDPOINT")},
		Transport: client.DefaultTransport,
		HeaderTimeoutPerRequest:time.Second,
	}
	err := checkEtcdConnect("gy",cfg)
	if err!=nil {
		log.Printf("connetcing to etcd failed")
		return
	}
	phandler := handlers.NewPromhandler(cfg)
	ahandler := handlers.NewAlerthandler(cfg)
	rhandler := handlers.NewRulehandler(cfg)
	ch := make(chan int)
	go handlers.Task(cfg,"prometheus",phandler,ch)
	go handlers.Task(cfg,"alert",ahandler,ch)
	go handlers.Task(cfg,"prometheus/resource_monitor",rhandler,ch)
	<-ch
}
