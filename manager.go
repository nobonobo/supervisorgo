package supervisorgo

import (
	"net/http"
	"sync"

	"github.com/kardianos/service"
	"github.com/nobonobo/jsonrpc"
)

var logger service.Logger

// SetLogger ...
func SetLogger(l service.Logger) {
	logger = l
}

// ConfigSet ...
type ConfigSet struct {
	ControlUri string
	Procs      []*Config
}

type Manager struct {
	procs map[string]*Process
	order []string
	wg    sync.WaitGroup
	done  chan struct{}
}

// NewManager ...
func NewManager(config *ConfigSet) *Manager {
	procs := map[string]*Process{}
	order := []string{}
	for _, c := range config.Procs {
		proc := New(c)
		name := proc.Name()
		if procs[name] != nil {
			logger.Warningf("duplicate name %s load skipped", name)
			continue
		}
		procs[name] = proc
		order = append(order, name)
	}
	return &Manager{procs: procs, order: order}
}

func (m *Manager) HTTPServe(path string) {
	server := jsonrpc.NewServer()
	server.Register(&Controller{manager: m})
	http.Handle(path, server)
}

func (m *Manager) run(proc *Process) {
	defer m.wg.Done()
	name := proc.Name()
	logger.Infof("start: %s", name)
	defer logger.Infof("terminated: %s", name)
	ech := proc.Start()
	defer proc.Stop()
	for {
		select {
		case <-m.done:
			return
		case err, ok := <-ech:
			if !ok {
				return
			}
			if err != nil {
				logger.Errorf("error %s: %s", name, err)
			}
		}
	}
}

func (m *Manager) Run(name string) {
	m.wg.Add(1)
	go m.run(m.procs[name])
}

func (m *Manager) Start(s service.Service) error {
	logger.Infof("start: supervisorgo")
	m.done = make(chan struct{})
	for _, name := range m.order {
		m.Run(name)
	}
	return nil
}

func (m *Manager) Stop(s service.Service) error {
	close(m.done)
	m.wg.Wait()
	logger.Infof("stop: supervisorgo")
	return nil
}
