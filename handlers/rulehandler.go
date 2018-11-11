package handlers

import (
	"github.com/coreos/etcd/client"
	"os"
	"log"
	"context"
	"encoding/json"
	"strings"
	"strconv"
	"html/template"
	"errors"
)


var (
	templates =[]string{"mysql","snmp","tomcat","cas_cvk","cas_vm","cas_cluster","k8sc","k8sn"}
)
type Rulehandler struct {
	cfg client.Config
}
type Parameter struct {
	Job string `json:"job_name"`
	Range string `json:"scrape_interval"`
	Type string `json:"metrics_path"`
}

func (h *Rulehandler)HandleEvents(resp *client.Response) error {
	if resp.Action =="set" || resp.Action =="update"{
		log.Printf("key: %v update",resp.Node.Key)
		return createRecordRules(ProcessData(resp))
	}
	if resp.Action == "delete"{
		log.Printf("key: %v delete",resp.Node.Key)
		delRuleFiles(h.cfg,resp)
	}
	return nil
}

func NewRulehandler(cfg client.Config) Handler {
	initRules(cfg)
	return &Rulehandler{cfg}
}
func initRules(cfg client.Config) {
	path := "/etc/prometheus/recordrules"
	err := os.RemoveAll(path)
	if err!=nil {
		log.Printf("remove rules error:%s",err.Error())
		return
	}
	err1 := os.MkdirAll(path,os.ModePerm)
	if err1!=nil {
		log.Printf("make dir error: %s",err1.Error())
		return
	}
	readEtcdInfo(cfg)
}
func readEtcdInfo(cfg client.Config) {
	c,err := client.New(cfg)
	if err!=nil {
		log.Printf("%s",err.Error())
		return
	}
	kapi := client.NewKeysAPI(c)
	resp1,err := kapi.Get(context.Background(),"/gy/prometheus/resource_monitor",nil)
	if err!=nil {
		return
	}else {
		for _,v := range resp1.Node.Nodes {
			v_scrape := []byte(v.Value)
			var param Parameter
			json.Unmarshal(v_scrape,&param)
			createRecordRules(param)
		}
	}

}
func createRecordRules(parameter Parameter) error{
	if !matchNode(parameter.Job) {
		return nil
	}
	parameter.Type =strings.TrimPrefix(parameter.Type,"/")
	rs := []rune(parameter.Range)
	times,_ := strconv.Atoi(string(rs[0:len(parameter.Range)-1]))
	ss := strconv.Itoa(times * 2)
	units := string(rs[len(parameter.Range)-1:len(parameter.Range)])
	parameter.Range = ss + units
	for _,v := range templates{
		if parameter.Type ==v {
			err := useRuleTemplate(parameter)
			if err!=nil {
				log.Printf("init rules error: %s",err.Error())
			}
			return err
		}else {
			continue
		}
	}
	return errors.New("no such template:"+parameter.Type)
}
func useRuleTemplate(parameter Parameter) error {
	if len(parameter.Range) >0 && len(parameter.Type) >0 && len(parameter.Job) > 0 {
		templ,err := template.ParseFiles("/ruletemplates/"+parameter.Type+".yml")
		if err!=nil {
			log.Printf("use rule template error:%s",err.Error())
			return err
		}
		dstFile,err := os.Create("/etc/prometheus/recordrules/"+parameter.Job+".yml")
		if err!=nil {
			log.Printf("create file error : %s",err.Error())
			return err
		}
		//defer dstFile.Close()
		err = templ.Execute(dstFile,parameter)
		if err!=nil {
			log.Printf("save rules error: %s",err.Error())
			return err
		}
		return nil
	}else {
		return errors.New("get parameters error")
	}
}

func delRuleFiles(cfg client.Config,resp *client.Response)  {
	pp := ProcessData(resp)
	fileName := pp.Job
	path := "/etc/parometheus/recordrules/"
	removeFile(path,fileName)
	return
}
func ProcessData(resp *client.Response) Parameter{
	if resp.Action == "set" || resp.Action == "update" {
		return JsonToStruct([]byte(resp.Node.Value))
	}else if resp.Action == "delete" {
		return JsonToStruct([]byte(resp.PrevNode.Value))
	}
	return Parameter{}
}
func JsonToStruct(v []byte) Parameter {
	var param Parameter
	json.Unmarshal(v,&param)
	return param
}
