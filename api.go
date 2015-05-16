package supervisorgo

import (
	"fmt"
	"time"
)

type Controller struct {
	manager *Manager
}

type Info struct {
	Name   string
	Since  time.Duration
	Status Status
}

func (c *Controller) Status(target string, reply *[]Info) error {
	if target == "" {
		for _, name := range c.manager.order {
			proc := c.manager.procs[name]
			*reply = append(*reply, Info{
				Name:   proc.Name(),
				Since:  proc.Since(),
				Status: proc.Status(),
			})
		}
	} else {
		proc := c.manager.procs[target]
		if proc == nil {
			return fmt.Errorf("unknown proc name: %s", target)
		}
		*reply = append(*reply, Info{
			Name:   proc.Name(),
			Since:  proc.Since(),
			Status: proc.Status(),
		})

	}
	return nil
}
