package handlers

import (
	"github.com/coreos/etcd/client"
	"log"
	"github.com/ghodss/yaml"
	yaml2 "gopkg.in/yaml.v2"
	"os"
	"github.com/prometheus/common/model"
	"prom-conf-gen/config"
)

type Promehandler struct {
	cfg client.Config
}

const (
	global    = "/gy/prometheus/global"
	ruleFiles = "/gy/prometheus/rule_files"
	alerting = "/gy/prometheus/alerting"
)

func (h *Promehandler) HandleEvents(resp *client.Response) error {
	if resp.Action == "set" || resp.Action == "update" || resp.Action == "delete" {
		c1, c2, c3, c4, c5,c6, err := getPrometheusConfig(h.cfg)
		if err != nil {
			log.Printf("error to get prometheus config : %s", err.Error())
			return err
		}
		err = configgen(c1, c2, c3, c4, c5,c6)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewPromhandler(cfg client.Config) Handler {
	initPrometheus(cfg)
	return &Promehandler{cfg}
}

func initPrometheus(cfg client.Config) {
	c1, c2, c3, c4, c5,c6, err := getPrometheusConfig(cfg)
	if err != nil {
		log.Printf("%s", err.Error())
		return
	}
	err = configgen(c1, c2, c3, c4, c5,c6)
	if err != nil {
		log.Printf("generate prometheus.yml error : %s", err.Error())
	}
}
func configgen(global config.GlobalConfig, rulefiles []string, scrapeconfigs []*config.ScrapeConfig,
	remotewrite []*config.RemoteWriteConfig, remoteread []*config.RemoteReadConfig,alertA config.AlertingConfig) error {
	newconf := config.Config{
		GlobalConfig:       global,
		AlertingConfig:     alertA,
		RuleFiles:          rulefiles,
		ScrapeConfigs:      scrapeconfigs,
		RemoteWriteConfigs: remotewrite,
		RemoteReadConfigs:  remoteread,
	}
	data, err := yaml2.Marshal(newconf)
	if err != nil {
		log.Printf("fail to marshal: %s", err.Error())
		return err
	}
	//if IsExists("c:/etc/prometheus/prometheus.yml") {
	//	err = removeFile("c:/etc/prometheus/", "prometheus.yml")
	//	if err != nil {
	//		log.Printf("remove prometheus.yml error:%s", err.Error())
	//		return err
	//	}
	//}
	saveFile("/etc/prometheus/", "prometheus.yml", data)
	return nil
}

func getPrometheusConfig(cfg client.Config) (config.GlobalConfig, []string, []*config.ScrapeConfig,
	[]*config.RemoteWriteConfig, []*config.RemoteReadConfig, config.AlertingConfig,error) {
	global_cfg := &config.GlobalConfig{}
	alert_addr := &config.AlertingConfig{}
	rule_cfg := &[]string{}
	s := []*config.ScrapeConfig{}
	Remote_write := []*config.RemoteWriteConfig{}
	Remote_read := []*config.RemoteReadConfig{}
	etcdstring := []string{global, ruleFiles,alerting}
	for _, v := range etcdstring {
		eValue, err := getEtcdValue(v, cfg)
		if err != nil {
			log.Printf("%s", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
		}
		yValue, err := yaml.JSONToYAML([]byte(eValue))
		if err != nil {
			log.Printf("%s", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
		}
		if v == global {
			yaml2.Unmarshal(yValue, &global_cfg)
		}
		if v == ruleFiles {
			yaml2.Unmarshal(yValue, &rule_cfg)
		}
		if v == alerting {
			yaml2.Unmarshal(yValue,&alert_addr)
		}
	}
	flag := os.Getenv("FLAG")
	// "write1 write2 read 是在prometheus工程集群的yaml文件中设置prom-gen镜像的参数"
	if flag == "" || flag == "write1" || flag == "write2" {
		kv, err := getEtcdValues("/gy/prometheus/resource_monitor", cfg)
		if err != nil {
			log.Printf("%s", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
		}
		for _, v := range kv {
			v_scrape := []byte(v)
			y_scrape, err := yaml.JSONToYAML(v_scrape)
			if err != nil {
				log.Printf("%s", err.Error())
				return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
			}
			scrape_cfg := &config.ScrapeConfig{}
			yaml2.Unmarshal(y_scrape, &scrape_cfg)
			s = append(s, scrape_cfg)
		}
	}
	//只读的prometheus，所有设轮询时间很长，自动超时，就不会从exporter去获取数据
	if flag == "read" {
		var interval model.Duration
		kv, err := getEtcdValues("/gy/prometheus/resource_monitor", cfg)
		if err == nil {
			interval, err = model.ParseDuration("365d")
		}
		if err != nil {
			log.Printf("err: %s\n", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
		}
		for _, v := range kv {
			v_scrape := []byte(v)
			y_scrape, err := yaml.JSONToYAML(v_scrape)
			if err != nil {
				log.Printf("%s", err.Error())
				return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
			}
			scrape_cfg := &config.ScrapeConfig{}
			yaml2.Unmarshal(y_scrape, &scrape_cfg)
			scrape_cfg.ScrapeInterval = interval
			s = append(s, scrape_cfg)
		}
	}

	kvrw, err := getEtcdValues("/gy/prometheus/remote_write", cfg)
	if err != nil {
		log.Printf("%s", err.Error())
		return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
	}
	for _, v := range kvrw {
		v_rw := []byte(v)
		y_rw, err := yaml.JSONToYAML(v_rw)
		if err != nil {
			log.Printf("%s", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr, err
		}
		rw_cfg := &config.RemoteWriteConfig{}
		yaml2.Unmarshal(y_rw, &rw_cfg)
		Remote_write = append(Remote_write, rw_cfg)
	}
	kvrr, err := getEtcdValues("/gy/prometheus/remote_read", cfg)
	if err != nil {
		log.Printf("%s", err.Error())
		return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr,err
	}
	for _, v := range kvrr {
		v_rr := []byte(v)
		y_rr, err := yaml.JSONToYAML(v_rr)
		if err != nil {
			log.Printf("%s", err.Error())
			return *global_cfg, *rule_cfg, s, Remote_write, Remote_read,*alert_addr,err
		}
		rr_cfg := &config.RemoteReadConfig{}
		yaml2.Unmarshal(y_rr, &rr_cfg)
		Remote_read = append(Remote_read, rr_cfg)
	}
	var retConfig config.GlobalConfig
	var retRules []string
	var retAlert config.AlertingConfig
	if global_cfg != nil {
		retConfig = *global_cfg
	}
	if rule_cfg != nil {
		retRules = * rule_cfg
	}
	if alert_addr !=nil {
		retAlert = * alert_addr
	}
	return retConfig, retRules, s, Remote_write, Remote_read,retAlert, nil
}

