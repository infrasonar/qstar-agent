[![CI](https://github.com/infrasonar/qstar-agent/workflows/CI/badge.svg)](https://github.com/infrasonar/qstar-agent/actions)
[![Release Version](https://img.shields.io/github/release/infrasonar/qstar-agent)](https://github.com/infrasonar/qstar-agent/releases)

# InfraSonar Linux Agent

Documentation: https://docs.infrasonar.com/collectors/agents/qstar/

## Environment variables

Environment                 | Default                       | Description
----------------------------|-------------------------------|-------------------
`STORAGE_PATH`              | `HOME/.infrasonar/`           | Path where files are stored _(not used when `ASSET_ID` is set)_.
`TOKEN`                     | _required_                    | Token used for authentication _(This MUST be a container token)_.
`ASSET_NAME`                | _none_                        | Initial Asset Name. This will only be used at the announce. Once the asset is created, `ASSET_NAME` will be ignored.
`ASSET_ID`                  | _none_                        | Asset Id _(If not given, the asset Id will be stored and loaded from file)_.
`API_URI`                   | https://api.infrasonar.com    | InfraSonar API.
`CHECK_QSTAR_INTERVAL`      | `300`                         | Interval in seconds for the `qstar` check.


## Installation

### Using Systemd

```bash
$ sudo touch /etc/systemd/system/infrasonar-qstar-agent.service
$ sudo chmod 664 /etc/systemd/system/infrasonar-qstar-agent.service
```

Using you favorite editor, add the content below to the file created

```
[Unit]
Description=InfraSonar Linux Agent
Wants=network.target

[Service]
Environment="TOKEN=<YOUR TOKEN HERE>"
# Environment="ASSET_ID=<YOUR ASSET ID>"
# Environment="STORAGE_PATH=<PATH_TO_STORE_ASSET_FILE>"
ExecStart=/usr/sbin/infrasonar-qstar-agent

[Install]
WantedBy=multi-user.target
```

Reload systemd

```bash
$ sudo systemctl daemon-reload
```

Install the service
```bash
$ sudo systemctl enable infrasonar-qstar-agent
```

You may want to start/stop or view the status
```bash
$ sudo systemctl start infrasonar-qstar-agent
$ sudo systemctl stop infrasonar-qstar-agent
$ sudo systemctl status infrasonar-qstar-agent
```

View logging:
```bash
$ journalctl -u infrasonar-qstar-agent
```
