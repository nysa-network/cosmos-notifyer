# Cosmos notifyer

Cosmos notifyer is an alerting and informing tool for cosmos-sdk based chain


# Features

:warning: Alerts

- [x] New :moneybag: & Lost :money_with_wings: delegations
- [x] Missing blocks and recovery
- [ ] RPCs are down

* Cosmos-notifyer can send alert into 

- [x] Discord
- [ ] Slack
- [ ] Telegram
- [ ] Phone number
- [ ] Homing pigeon 

## Installation

to install docker -> [get.docker.com](https://get.docker.com)

* using [docker](https://docker.com)


```bash
$ cp config.example.yml config.yml

# edit the config
$ vi config.yml

# start the container
$ docker-compose up -d
```

## Misc

This tool is inspired by [blockpane/tenderduty](https://github.com/blockpane/tenderduty)
