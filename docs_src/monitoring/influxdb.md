# InfluxDB

## Overview
InfluxDB is an open-source time series database that is commonly combined with Telegraf and Grafana in monitoring implementations. InfluxDB is a preconfigured data source for the Grafana UI within AiCSD, which enables the Telegraf system metrics stored within InfluxDB to automatically be displayed using Grafana. InfluxDB is incorporated in the AiCSD monitoring stack through the use of an output plugin specified within the Telegraf configuration. 

System metrics are collected by Telegraf and stored within an InfluxDB bucket. The `systemHealthMonitoring` bucket is the InfluxDB storage bucket used for system health metrics. 

## InfluxDB 2.0
InfluxDB version 2.0 is used along with Flux-styled queries. Flux is InfluxData's functional data scripting language designed for querying, analyzing, and acting on data. Grafana enables Flux-styled queries to interact with the InfluxDB data for visualizations.

## Visit InfluxDB UI
Navigate to [http://localhost:8086](http://localhost:8086)

## InfluxDB Authentication
InfluxDB version 2.0 provides bootstrap functionality to enable an initial admin user, organization, and storage bucket. Additional environment variables are used to configure the setup logic found in the configuration file at the root directory of this project. The environment configuration is contained in the `.env` file.

!!! Note
    For proper authentication, necessary to enable monitoring, change the placeholder values for password and token before starting the services. These are contained in the `.env` file. 
    
    Open the `.env` file and modify the values for `DOCKER_INFLUXDB_INIT_PASSWORD` and `DOCKER_INFLUXDB_INIT_ADMIN_TOKEN`. These variables may be found at the bottom of the file under the `Monitoring` section and contain the placeholder value of `<CHANGE_ME!>`. 

### Login
The initial username is `admin` by default. The password to login is the password set in the `.env` file before starting the monitoring services. If necessary, create additional users upon the admin initial login.

## View Telegraf Data Bucket
To view the Telegraf data bucket, use the nav bar on the left.

Click **Data > Buckets > systemHealthMonitoring**.

## Resources

- [InfluxDB](https://www.influxdata.com/)
- [InfluxDB Docs](https://docs.influxdata.com/influxdb/v2.4/get-started/)



BSD-3 License: See [License](../LICENSE.md).
