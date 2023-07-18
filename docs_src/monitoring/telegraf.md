# Telegraf

## Overview

Telegraf acts as a StatsD agent to collect host system metrics. It includes plugins to collect these metrics, many of which are enabled by default. Telegraf may be deployed to ingest data from multiple input sources and deliver that data to numerous output sources. In AiCSD, Telegraf collects system metrics on utilization (CPU, disk, memory, etc.) and writes data to an InfluxDB output plugin.

## Telegraf Authentication

Telegraf authenticates with InfluxDB via the `DOCKER_INFLUXDB_INIT_ADMIN_TOKEN` value specified in the `.env` file. The bootstrapping of the `DOCKER_INFLUXDB_INIT_ADMIN_TOKEN` enables a seamless authentication experience between InfluxDB and Telegraf. Telegraf metrics are then written to InfluxDB. 

## Resources

- [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/)
- [GitHub](https://github.com/influxdata/telegraf)
- [Telegraf Docs](https://github.com/influxdata/telegraf/tree/master/docs)

BSD-3 License: See [License](../LICENSE.md).