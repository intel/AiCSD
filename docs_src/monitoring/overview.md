# Overview

The open-source tools, [Telegraf](../monitoring/telegraf.md), [InfluxDB](../monitoring/influxdb.md) and [Grafana](../monitoring/grafana.md) (TIG), comprise the TIG stack, a tech stack commonly used for collecting, storing, and displaying time series data. AiCSD uses the TIG stack in the implementation of the health monitoring feature.

Figure 1 illustrates the data flow and component relationships in the TIG stack.

<figure class="figure-image">
<img src="..\images\AggregateMetrics.png" alt="Figure 1: Monitoring< Architecture">
<figcaption>Figure 1: Monitoring Architecture</figcaption>
</figure>

!!! Note
    It isn't necessary to download the tools of the TIG stack separately as they are included in the monitoring feature of AiCSD. However, to find out more about each tool, see the Resources section for each of the tools.

## Enable System Health Monitoring
To enable monitoring services, update authentication information in the `.env` file found within the root directory, as described in [Run the Services](#run-the-services). This enables Telegraf to send system metrics to InfluxDB. 

!!! Note
    While some out-of-the-box security measures were taken for the single-user environment, this repository contains the base implementation for monitoring. It is not configured for complex projects or deployment-specific security concerns.

### Run the Services
To update the authentication information in the `.env` file:
 
1. Open the `.env` file found at the root level of this project.
2. Scroll down to the bottom of the file to the `Monitoring` section.
3. Find the `DOCKER_INFLUXDB_INIT_PASSWORD` variable and update the value with a password at least 8 characters in length. Be sure to remove the `<>` characters.
4. Find the `DOCKER_INFLUXDB_INIT_ADMIN_TOKEN` variable, update the token value and remove the `<>` characters.
5. Start Telegraf, InfluxDB, and Grafana:

    ```bash
      make run-monitor
    ```

    This command creates a monitoring Docker network, bringing up the services necessary for monitoring purposes.

### View System Health
To view the default system health dashboard on the Grafana UI, open a browser to [http://localhost:3001](http://localhost:3001).

INTEL CONFIDENTIAL: See [License](../LICENSE.md).
