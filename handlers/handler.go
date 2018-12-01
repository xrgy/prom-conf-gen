package handlers

import (
	"github.com/coreos/etcd/client"
	"os"
	"log"
	"io/ioutil"
	"errors"
	"context"
	"crypto/md5"
)

type Handler interface {
	HandleEvents(client *client.Response) error
}

//A task to watch a key and do something when key update
func Task(cfg client.Config,key string, handler Handler,ch chan<-int)  {
	defer func() {
		ch <- 1
		close(ch)
	}()
	cc,err := client.New(cfg)
	if err!=nil {
		log.Printf("%s",err.Error())
		return
	}
	kapi := client.NewKeysAPI(cc)
	watcher := kapi.Watcher("/gy/"+key,&client.WatcherOptions{
		Recursive:true,
	})
	for {
		res,err := watcher.Next(context.Background())
		if err!=nil {
			log.Printf("error watch workers:%s",err.Error())
			break
		}
		handlerErr := handler.HandleEvents(res)
		if handlerErr!=nil {
			log.Printf("error handler event: %s",handlerErr.Error())
			continue
		}
		}
}

//save file in given path
func saveFile(path string, filename string, data []byte) error{
	err := os.MkdirAll(path,os.ModePerm)
	if err!=nil {
		log.Printf("make dir "+path+" error:%s",err.Error())
		return err
	}
	err = ioutil.WriteFile(path+filename,data,0644)
	if err !=nil {
		log.Printf("save file error:%s",err.Error())
	}
	return nil
}

//remove file in a given path
func removeFile(path string, filename string) error{
	err := os.Remove(path + filename)
	if err!=nil {
		log.Printf("remove file error: %s",err.Error())
		return err
	}
	return nil
}

func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

//get all keys under the directory and return a map. if there is error,return nil
func getEtcdValues(dir string, cfg client.Config) (map[string]string,error) {
	c, err := client.New(cfg)
	if err != nil {
		log.Printf("get etcd client error:%s", err.Error())
		return nil, err
	}
	m := make(map[string]string)
	kapi := client.NewKeysAPI(c)
	retryTimes := 3
	for count := 0; count < retryTimes; count++ {
		resp, err := kapi.Get(context.Background(), dir, nil)
		if err != nil {
			log.Printf(dir+" retry time %v", count+1)
			continue
		} else {
			for _,v := range resp.Node.Nodes{
				m[v.Key] = v.Value
			}
			return m,nil
		}
	}
	return nil,err
}

//get the value for the given key
func getEtcdValue(key string, cfg client.Config) (string, error) {
	c, errs := client.New(cfg)
	if errs != nil {
		log.Printf("get etcd client error:%s", errs.Error())
		return "", errs
	}
	kapi := client.NewKeysAPI(c)
	retryTimes := 3
	for count := 0; count < retryTimes; count++ {
		resp, err := kapi.Get(context.Background(), key, nil)
		if err != nil {
			log.Printf("retry time %v", count+1)
			errs = err
			continue
		} else {
			return getRespValue(resp)
		}
	}
	return "", errs
}

//get the value of 	the updated key from etcd response
func getRespValue(resp *client.Response) (string, error) {
	if resp != nil {
		if resp.Node != nil {
			return resp.Node.Value, nil
		} else {
			return "", errors.New("nil of etcd response node")
		}
	} else {
		return "", errors.New("nil etcd response")
	}
}

func matchNode(uuid string) bool {
	flag := os.Getenv("FLAG")
	if flag == "write1" || flag == "write2" {
		mod := sum64(md5.Sum([]byte(uuid))) % 2
		if flag=="write1" && mod ==0 {
			return true
		}
		if flag=="write2" && mod ==1 {
			return true
		}
		return false
	}else {
		if flag == "read" {
			return false
		}
	}
	return true
}
func sum64(hash [md5.Size]byte) uint64 {
	var s uint64
	for i,b := range hash{
		shift := uint64((md5.Size-i-1) * 8)
		s |= uint64(b) << shift
	}
	return s
}
