# File Receiver Gateway Microservice

## Overview
The File Receiver Gateway microservice responds to TransmitJob and TransmitFile API endpoints. On startup, it queries for unprocessed job events. The File Receiver Gateway writes files sent to it from the File Sender OEM to the output directory specified in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-receiver-gateway/res/configuration.toml) file.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](./ms-job-repository.md)
- [Task Launcher](./as-task-launcher.md)

## Swagger Documentation

<swagger-ui src="./api-definitions/ms-file-receiver-gateway.yaml"/>

## Next up

[Deep Dive into the Services - Task Launcher](./as-task-launcher.md)

INTEL CONFIDENTIAL: See [License](../LICENSE.md).
