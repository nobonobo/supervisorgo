# supervisorgo

supervisord like golang implement

## feature

- multiple process management.
- cross platform (windows/linux/os-x) supported.
- remote controll utility bundled.


## install

```
go get -u github.com/supervisorgo/cmd/...
```

## configuration json file

default-path = "execute-parent-dir/supervisorgo.json"

sample
```
{
	"ControlUri": "unix:./supervisorgo.sock",
	"Procs": [
		{
			"Name": "builder",
			"DisplayName": "Go Builder",
			"Description": "Run the Go Builder",

			"Dir": "/tmp",
			"Exec": "bash",
			"Args": ["-c","while true; do echo hello; sleep 5; exit 1; done"],
			"Env": [
			],

			"Stderr": "sample-err.log",
			"Stdout": "sample-out.log",
			"Interval": 1000,
			"Retry": 3
		}
	]
}
```

## usage

### foreground mode(for debug run)
```
supervisorgo -c config.json
```

### supervisorgo daemon control
```
supervisorgo start
supervisorgo stop
supervisorgo restart
sudo supervisorgo install   # daemon install to system
sudo supervisorgo uninstall # daemon uninstall from system
```

### usage for supervisorgoctl

supervisorctl [-c unix:./supervisorgo.sock] subcommand [args...]

```
supervisorctl status
(list procs)
supervisorctl status target-name
supervisorctl start target-name
supervisorctl stop target-name
```
