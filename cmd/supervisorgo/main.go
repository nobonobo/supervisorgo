package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"

	"github.com/nobonobo/jsonrpc"
	"github.com/nobonobo/supervisorgo"
)

var logger service.Logger

func getConfigPath() (string, error) {
	fullexecpath, err := osext.Executable()
	if err != nil {
		return "", err
	}
	dir, execname := filepath.Split(fullexecpath)
	ext := filepath.Ext(execname)
	name := execname[:len(execname)-len(ext)]
	return filepath.Join(dir, name+".json"), nil
}

func getConfig(path string) (*supervisorgo.ConfigSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := new(supervisorgo.ConfigSet)

	r := json.NewDecoder(f)
	err = r.Decode(&conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func main() {
	configPath := ""
	flag.StringVar(&configPath, "c", "", "config json-file path")
	flag.Parse()

	if configPath == "" {
		c, err := getConfigPath()
		if err != nil {
			log.Fatal(err)
		}
		configPath = c
	}

	svcConfig := &service.Config{
		Name:        "supervisorgo",
		DisplayName: "SuperVisorGo",
		Description: "process bootstrap daemon",
		Option: service.KeyValue{
			"UserService": true,
			"RunAtLoad":   true,
		},
	}

	conf, err := getConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	if conf.ControlUri == "" {
		conf.ControlUri = "unix:./supervisorgo.sock"
	}
	u, err := url.Parse(conf.ControlUri)
	if err != nil {
		log.Fatal(err)
	}
	m := supervisorgo.NewManager(conf)
	s, err := service.New(m, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}
	supervisorgo.SetLogger(logger)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errs {
			if err != nil {
				log.Print(err)
			}
		}
	}()
	defer wg.Wait()
	defer close(errs)

	if len(flag.Args()) != 0 {
		if err := service.Control(s, flag.Arg(0)); err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		log.Printf("%s succeeded", flag.Arg(0))
		return
	}

	os.Exit(run(s, m, u))
}

func run(s service.Service, m *supervisorgo.Manager, u *url.URL) int {
	var err error
	var l net.Listener
	path := jsonrpc.DefaultRPCPath
	switch u.Scheme {
	case "unix":
		l, err = net.Listen(u.Scheme, u.Opaque+u.Path)
		if l != nil {
			defer os.Remove(u.Opaque + u.Path)
		}
	case "http":
		l, err = net.Listen("tcp", u.Host)
		path = u.Path
	default:
		err = fmt.Errorf("not supported scheme: %s", u.Scheme)
	}
	if err != nil {
		log.Println(err)
		return 1
	}
	m.HTTPServe(path)
	go http.Serve(l, nil)

	err = s.Run()
	if err != nil {
		log.Println(err)
		return 1
	}
	return 0
}
