# Grafana

## Overview
Grafana is an open-source analytics and interactive visualization web application. AiCSD uses Grafana dashboards to display system metrics captured by Telegraf and stored within InfluxDB. 

The landing page of the Grafana UI displays the default system health dashboard, `Telegraf Metrics Dashboard`.

## Visit Grafana UI
Navigate to [http://localhost:3001](http://localhost:3001)

## Grafana Authentication
Authentication is currently disabled for Grafana. Access to the dashboards is enabled automatically for a more seamless user experience.

## Dashboard and Metrics
The dashboard displays the system health metrics captured by Telegraf and stored within InfluxDB.

This dashboard refreshes every 5 seconds and is preconfigured to use InfluxDB and the `systemHealthMonitoring` bucket for data. 

Adjust any of the fields at the top to better customize the dashboard.

## Resources

- [General Information: Grafana](https://grafana.com/oss/grafana/)
- [Grafana Docs](https://grafana.com/docs/)

BSD 3-Clause License: See [License](../LICENSE.md).