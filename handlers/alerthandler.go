package handlers

import (
	"github.com/coreos/etcd/client"
	"log"
	"regexp"
	"strings"
)

type Alerthandler struct {
	cfg client.Config
}

func (h *Alerthandler)HandleEvents(resp *client.Response) error  {
	if resp.Action =="set" || resp.Action =="update"{
		log.Printf("key: %v update",resp.Node.Key)
		rule,err :=getRespValue(resp)
		if err!=nil {
			log.Printf("error get alertules: %s",err.Error())
			return err
		}
		uuid := parseAlert([]byte(rule))
		if uuid !="" {
			if !matchNode(uuid) {
				return nil
			}
		}
		path := strings.Split(resp.Node.Key,"/")
		id := path[3]
		err = saveFile("/etc/prometheus/rules/",id+".yml",[]byte(rule))
		if err!=nil {
			return err
		}
	}
	if resp.Action =="delete" {
		log.Printf("key: %v delete",resp.Node.Key)
		path := strings.Split(resp.Node.Key,"/")
		id := path[3]
		err := removeFile("/etc/prometheus/rules/",id+".yml")
		if err!=nil {
			return err
		}
	}
	return nil
}

func NewAlerthandler(cfg client.Config) Handler {
	initAlert(cfg)
	return &Alerthandler{cfg}
}
func initAlert(cfg client.Config) {
	kv,err := getEtcdValues("/gy/alert",cfg)
	if err!=nil {
		log.Printf("error get etcd value:%s",err.Error())
	}
	for k,v := range kv {
		v_alert_value := []byte(v)
		uuid := parseAlert([]byte(v_alert_value))
		if uuid !="" {
			if !matchNode(uuid) {
				continue
			}
		}
		path := strings.Split(k,"/")
		id := path[3]
		err := saveFile("/etc/prometheus/rules/",id+".yml",v_alert_value)
		if err!=nil {
			log.Printf("error to save alertrule%s",err.Error())
		}
	}
}

//find instance_uuid in alert rule
func parseAlert(rule []byte) string {
	uuid := regexp.MustCompile(`instance_id="(?P<uuid>\S+)"`)
	founduuid := uuid.FindStringSubmatch(strings.TrimSpace(string(rule)))
	if len(founduuid)>1 {
		return founduuid[1]
	}else {
		log.Printf("can not parse instance_id in rule:%s",string(rule))
		return ""
	}
}