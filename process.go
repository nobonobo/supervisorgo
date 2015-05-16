package supervisorgo

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

/* TODO:

   - プロセスのステータスを取得する方法を確保。
     - STANDBY
     - RUNNING
     - STOPPING
     - STOPPED
     - RETRYWAIT
   - 停止シグナルを投げる、タイムアウトでSIGKILLとする。
*/

// Config ...
type Config struct {
	Name, Description string
	Dir               string
	Exec              string
	Args              []string
	Env               []string
	Stderr, Stdout    string
	Retry             int
	StopSignal        string
	Interval          int
}

// Status ...
type Status int

const (
	STOPPED Status = iota
	STANDBY
	RUNNING
	STOPPING
	RETRYWAIT
)

func (s Status) String() string {
	switch s {
	case STOPPED:
		return "STOPPED"
	case STANDBY:
		return "STANDBY"
	case RUNNING:
		return "RUNNING"
	case STOPPING:
		return "STOPPING"
	case RETRYWAIT:
		return "RETRYWAIT"
	}
	return ""
}

// Process ...
type Process struct {
	config   *Config
	cmdMu    sync.RWMutex
	cmd      *exec.Cmd
	statusMu sync.RWMutex
	status   Status
	first    time.Time
	begin    time.Time
	retry    int
}

// New ...
func New(config *Config) *Process {
	return &Process{
		config: config,
	}
}

// Name ...
func (p *Process) Name() string {
	if len(p.config.Name) > 0 {
		return p.config.Name
	}
	return p.config.Exec
}

// First ...
func (p *Process) First() time.Time {
	return p.first
}

// Since ...
func (p *Process) Since() time.Duration {
	return time.Since(p.begin)
}

// Retry ...
func (p *Process) Retry() int {
	return p.retry
}

// Cmd ...
func (p *Process) Cmd() *exec.Cmd {
	p.cmdMu.RLock()
	defer p.cmdMu.RUnlock()
	return p.cmd
}

// Status ...
func (p *Process) Status() Status {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status
}

func (p *Process) setStatus(s Status) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()
	p.status = s
}

func (p *Process) setup() error {
	fullExec, err := exec.LookPath(p.config.Exec)
	if err != nil {
		return fmt.Errorf("Failed to find executable %q: %s", p.config.Exec, err)
	}

	cmd := exec.Command(fullExec, p.config.Args...)
	cmd.Dir = p.config.Dir
	cmd.Env = append(os.Environ(), p.config.Env...)

	var stdout, stderr *os.File
	if p.config.Stderr != "" {
		stderr, err = os.OpenFile(p.config.Stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("stderr open failed %q: %s", p.config.Stderr, err)
		}
		cmd.Stderr = stderr
	}
	if p.config.Stdout != "" {
		if p.config.Stdout == p.config.Stderr {
			cmd.Stdout = cmd.Stderr
		} else {
			stdout, err = os.OpenFile(p.config.Stdout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				if stderr != nil {
					stderr.Close()
				}
				return fmt.Errorf("stdout open failed %q: %s", p.config.Stdout, err)
			}
			cmd.Stdout = stdout
		}
	}
	p.cmdMu.Lock()
	p.kill(p.cmd)
	p.cmd = cmd
	p.cmdMu.Unlock()
	return nil
}

func (p *Process) run() error {
	p.begin = time.Now()
	if p.first.IsZero() {
		p.first = p.begin
	}
	cmd := p.Cmd()
	err := cmd.Run()
	if cmd.Stdout != nil {
		if fp, ok := cmd.Stdout.(*os.File); ok {
			fp.Close()
		}
	}
	if cmd.Stderr != nil {
		if fp, ok := cmd.Stderr.(*os.File); ok {
			fp.Close()
		}
	}
	return err
}

func (p *Process) kill(cmd *exec.Cmd) error {
	if cmd == nil {
		return nil
	}
	if cmd.ProcessState == nil || cmd.ProcessState.Exited() == false {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("kill failed: %s", err)
			}
		}
	}
	return nil
}

func (p *Process) isStopping() (res bool) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()
	res = p.status == STOPPING
	if !res {
		p.status = RETRYWAIT
	}
	return res
}

// Start ...
func (p *Process) Start() <-chan error {
	p.retry = 0
	p.first = time.Time{}
	ech := make(chan error, 1)
	if p.Status() != STOPPED {
		ech <- fmt.Errorf("already running: %s", p.Name())
		close(ech)
		return ech
	}
	p.setStatus(STANDBY)
	go func() {
		defer p.setStatus(STOPPED)
		defer close(ech)
		for {
			if err := p.setup(); err != nil {
				ech <- err
				return
			}
			if !p.first.IsZero() {
				if p.retry >= p.config.Retry {
					ech <- fmt.Errorf("retry over")
					return
				}
				p.retry++
			}
			p.setStatus(RUNNING)
			if err := p.run(); err != nil {
				if p.Status() == STOPPED {
					return
				}
				ech <- err
			}
			if p.isStopping() {
				return
			}
			time.Sleep(time.Duration(p.config.Interval * int(time.Microsecond)))
		}
	}()
	return ech
}

// Stop ...
func (p *Process) Stop() error {
	if p.Status() != RUNNING {
		return nil
	}
	p.setStatus(STOPPING)
	defer p.setStatus(STOPPED)
	cmd := p.Cmd()
	if err := p.kill(cmd); err != nil {
		return err
	}
	return cmd.Wait()
}
