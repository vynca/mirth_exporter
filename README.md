# Mirth Exporter

Export Mirth Connect channel statistics to Prometheus.

To run it:
```bash
go build
./mirth_exporter [flags]
```

## Exported Metrics
| Metric | Meaning | Labels |
| ------ | ------- | ------ |
| mirth_up | Was the last Mirth CLI query successful | |
| mirth_channels_deployed | How many channels are deployed | |
| mirth_channels_started | How many of the deployed channels are started | |
| mirth_messages_received | How many messages have been received | channel |
| mirth_messages_filtered | How many messages have been filtered | channel |
| mirth_messages_queued | How many messages are currently queued | channel |
| mirth_messages_sent | How many messages have been sent | channel |
| mirth_messages_errored | How many messages have errored | channel |

## Flags
```bash
./mirth_exporter --help
```

| Flag | Description | Default |
| ---- | ----------- | ------- |
| log.level | Logging level | `info` |
| mccli.config-path | Path to properties file for Mirth Connect CLI | `./mirth-cli-config.properties` |
| mccli.jar-path | Path to jar file for Mirth Connect CLI | `./mirth-cli-launcher.jar` |
| web.listen-address | Address to listen on for telemetry | `:9140` |
| web.telemetry-path | Path under which to expose metrics | `/metrics` |

## Notice

This exporter is inspired by the [consul_exporter](https://github.com/prometheus/consul_exporter)
and has some common code. Any new code here is Copyright &copy; 2016 Vynca, Inc. See the included
LICENSE file for terms and conditions.
