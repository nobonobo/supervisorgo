package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/nobonobo/jsonrpc"
	"github.com/nobonobo/supervisorgo"
)

var (
	flags = struct {
		dial string
	}{}

	status = template.Must(template.New("").Parse(
		"{{.Name}}\t{{.Status}}\t{{.Since}}\n",
	))
)

func init() {
	flag.StringVar(&flags.dial, "c", "unix:./supervisorgo.sock", "dial uri to connect")
	flag.Parse()
}

func main() {
	os.Exit(run())
}

func run() int {
	controller := jsonrpc.NewClient(flags.dial, nil).Get("Controller")
	switch flag.Arg(0) {
	case "status":
		target := ""
		if flag.NArg() > 1 {
			target = flag.Arg(1)
		}
		reply := []supervisorgo.Info{}
		if err := controller.Call("Status", target, &reply); err != nil {
			log.Println(err)
			return 1
		}
		for _, v := range reply {
			status.Execute(os.Stdout, v)
		}
	case "start":
		if flag.NArg() < 2 {
			log.Println(fmt.Errorf("argument target proc-name needed"))
			return 1
		}
		target := flag.Arg(1)
		reply := supervisorgo.STOPPED
		if err := controller.Call("Start", target, &reply); err != nil {
			log.Println(err)
			return 1
		}
	case "stop":
		if flag.NArg() < 2 {
			log.Println(fmt.Errorf("argument target proc-name needed"))
			return 1
		}
		target := flag.Arg(1)
		reply := supervisorgo.STOPPED
		if err := controller.Call("Stop", target, &reply); err != nil {
			log.Println(err)
			return 1
		}

	default:
		log.Println(fmt.Errorf("unknown sub command: %s", flag.Arg(0)))
		return 1
	}
	return 0
}
