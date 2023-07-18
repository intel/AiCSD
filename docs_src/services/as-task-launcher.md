# Task Launcher Application Service

## Overview
The Task Launcher microservice manages and launches tasks for jobs to be executed on the Pipeline Simulator, Geti pipelines, or BentoML pipelines.

## Dependencies
This application service is dependent on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis from EdgeX](https://docs.edgexfoundry.org/2.3/microservices/core/database/Ch-Redis/)
- [Redis for EdgeX MessageBus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [Job Repostiory](./ms-job-repository.md)
- [File Sender Gateway](./as-file-sender-gateway.md)

!!! Note
    The same Redis implementation is used both for the database and the Publish/Subscribe Message Broker needed for the EdgeX Message Bus.

## Configuration
Change task launcher configurations in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/as-task-launcher/res/configuration.toml) file. Configuration options can also be changed when the service is running by using [Consul](http://localhost:8500/ui/dc1/kv/edgex/appservices/2.0/as-task-launcher/ApplicationSettings/).

!!! Note 
    For changes to take effect, the service must be restarted. If changes are made to the configuration.toml file, the service must be stopped, rebuilt, and started again.

- **RetryWindow:** Determines how often a job should be resent to the pipeline for processing
- **DeviceProfileName:** Indicates the device profile information for the pipeline to consume
- **DeviceName:** Indicates the device name for the pipeline to consume


## Swagger Documentation

<swagger-ui src="./api-definitions/as-task-launcher.yaml"/>

## Next up

[Deep Dive into the Services - Pipeline Simulator](./as-pipeline-sim.md)

BSD 3-Clause License: See [License](../LICENSE.md).
