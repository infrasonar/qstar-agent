[![CI](https://github.com/infrasonar/qstar-agent/workflows/CI/badge.svg)](https://github.com/infrasonar/qstar-agent/actions)
[![Release Version](https://img.shields.io/github/release/infrasonar/qstar-agent)](https://github.com/infrasonar/qstar-agent/releases)

# InfraSonar QStar Agent

Documentation: https://docs.infrasonar.com/collectors/agents/qstar/

## Environment variables

Environment                 | Default                       | Description
----------------------------|-------------------------------|-------------------
`CONFIG_PATH`       		| `/etc/infrasonar` 			| Path where configuration files are loaded and stored _(note: for a user, the `$HOME` path will be used instead of `/etc`)_
`TOKEN`                     | _required_                    | Token used for authentication _(This MUST be a container token)_.
`ASSET_NAME`                | _none_                        | Initial Asset Name. This will only be used at the announce. Once the asset is created, `ASSET_NAME` will be ignored.
`ASSET_ID`                  | _none_                        | Asset Id _(If not given, the asset Id will be stored and loaded from file)_.
`API_URI`                   | https://api.infrasonar.com    | InfraSonar API.
`SKIP_VERIFY`				| _none_						| Set to `1` or something else to skip certificate validation.
`CHECK_QSTAR_INTERVAL`      | `300`                         | Interval in seconds for the `qstar` check.


## Build
```
CGO_ENABLED=0 go build -o qstar-agent
```

## Installation

Download the latest release:
```bash
$ wget https://github.com/infrasonar/qstar-agent/releases/download/v0.1.0/qstar-agent
```

> _The pre-build binary is build for the **linux-amd64** platform. For other platforms build from source using the command:_ `CGO_ENABLED=0 go build -o qstar-agent`

Ensure the binary is executable:
```
chmod +x qstar-agent
```

Copy the binary to `/usr/sbin/infrasonar-qstar-agent`

```
$ sudo cp qstar-agent /usr/sbin/infrasonar-qstar-agent
```

### Using Systemd

```bash
$ sudo touch /etc/systemd/system/infrasonar-qstar-agent.service
$ sudo chmod 664 /etc/systemd/system/infrasonar-qstar-agent.service
```

**1. Using you favorite editor, add the content below to the file created:**

```
[Unit]
Description=InfraSonar QStar Agent
Wants=network.target

[Service]
EnvironmentFile=/etc/infrasonar/qstar-agent.env
ExecStart=/usr/sbin/infrasonar-qstar-agent

[Install]
WantedBy=multi-user.target
```

**2. Create the file `/etc/infrasonar/qstar-agent.env` with at least:**

```
TOKEN=<YOUR TOKEN HERE>
```

Optionaly, add environment variable to the `qstar-agent.env` file for settings like `ASSET_ID` or `CONFIG_PATH` _(see all [environment variables](#environment-variables) in the table above)_.

**3. Reload systemd:**

```bash
$ sudo systemctl daemon-reload
```

**4. Install the service:**

```bash
$ sudo systemctl enable infrasonar-qstar-agent
```

**Finally, you may want to start/stop or view the status:**
```bash
$ sudo systemctl start infrasonar-qstar-agent
$ sudo systemctl stop infrasonar-qstar-agent
$ sudo systemctl status infrasonar-qstar-agent
```

**View logging:**
```bash
$ journalctl -u infrasonar-qstar-agent
```

**mmparam not found**
If the `mmparam` command is not found, this might be a problem with the path settings. As a solution a short link can be created:

```
ln -s /opt/QStar/bin/mmparam /usr/sbin/mmparam
```

