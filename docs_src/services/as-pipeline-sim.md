# Pipeline Simulator Application Service

## Overview
The Pipeline Simulator is an alternative to a third party machine learning(ML) platform for development and integration purposes.
It receives the `Launch` event and reading via the Message Bus, creates a simple copy of the input file as the output
file(s). It then calls the Job Repo and Pipeline Status endpoints.

The Pipeline Simulator also extends its functionality to support the Geti and BentoML sample ML pipelines.
It receives an MQTT message to execute a Geti or BentoML sample pipelines. It also queries the OpenVINO Model Server(OVMS) to retrieve the available models and populate them as Geti or BentoML pipelines in the dropdown for creating a new task via the UI.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [Job Repository](./ms-job-repository.md)
- [Task Launcher](./as-task-launcher.md)

## Swagger Documentation

<swagger-ui src="./api-definitions/as-pipeline-sim.yaml"/>

## Next up

[Deep Dive into the Services - File Sender Gateway](./as-file-sender-gateway.md)

BSD-3 License: See [License](../LICENSE.md).
