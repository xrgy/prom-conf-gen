package handlers

import (
	"github.com/coreos/etcd/client"
	"log"
	"regexp"
	"strings"
	"github.com/prometheus/common/model"
	"github.com/ghodss/yaml"
	yaml2 "gopkg.in/yaml.v2"
)

type Alerthandler struct {
	cfg client.Config
}

type RuleGroup struct {
	Name     string         `yaml:"name"`
	Interval model.Duration `yaml:"interval,omitempty"`
	Rules    []Rule         `yaml:"rules"`
}

// Rule describes an alerting or recording rule.
type Rule struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr"`
	For         model.Duration    `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

func (h *Alerthandler)HandleEvents(resp *client.Response) error  {
	if resp.Action =="set" || resp.Action =="update"{
		log.Printf("key: %v update",resp.Node.Key)
		rule,err :=getRespValue(resp)
		if err!=nil {
			log.Printf("error get alertules: %s",err.Error())
			return err
		}
		/*uuid := parseAlert([]byte(rule))
		if uuid !="" {
			if !matchNode(uuid) {
				return nil
			}
		}*/
		rule_groups := &RuleGroups{}
		yValue, err := yaml.JSONToYAML([]byte(rule))
		path := strings.Split(resp.Node.Key,"/")
		id := path[3]
		yaml2.Unmarshal(yValue, &rule_groups)
		data, err := yaml2.Marshal(*rule_groups)
		err = saveFile("/etc/prometheus/rules/",id+".yml",data)
		//cmd := exec.Command("/etc/promtool"," ","update rules /etc/prometheus/rules/"+id+".yml")
		//ret,_ := cmd.Output()
		//s := string(ret)
		//log.Printf("exec command /etc/promtool "+s)
		if err!=nil {
			log.Printf("error to save alertrule%s",err.Error())
		}
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
		rule_groups := &RuleGroups{}
		yValue, err := yaml.JSONToYAML([]byte(v))
		//v_alert_value := []byte(v)
		/*uuid := parseAlert([]byte(v_alert_value))
		if uuid !="" {
			if !matchNode(uuid) {
				continue
			}
		}*/
		//saveFile("c:/etc/prometheus/rules/","aaa.yml",yValue)  也是可以的 只是每个字段的顺序不一样
		path := strings.Split(k,"/")
		id := path[3]
		err = removeFile("/etc/prometheus/rules/",id+".yml")
		if err!=nil {
			log.Printf("remove rules.rules error:%s",err.Error())
		}
		yaml2.Unmarshal(yValue, &rule_groups)
		data, err := yaml2.Marshal(*rule_groups)
		err = saveFile("/etc/prometheus/rules/",id+".yml",data)
		/*cmd := exec.Command("c:/etc/promtool"," ","update rules /etc/prometheus/rules/"+id+".rules")
		ret,_ := cmd.Output()
		s := string(ret)
		log.Printf("exec command /etc/promtool "+s)*/
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