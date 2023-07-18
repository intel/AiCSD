# File Sender OEM Microservice

## Overview
The File Sender OEM microservice listens for events and sends files from those events. On startup, it queries for
unprocessed job events. The File Sender OEM sends files received from the data organizer to the 
File Receiver Gateway. The configuration information is set in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-sender-oem/res/configuration.toml) file.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](./ms-job-repository.md)
- [File Receiver Gateway](./ms-file-receiver-gateway.md)

## Swagger Documentation

<swagger-ui src="./api-definitions/ms-file-sender-oem.yaml"/>

## Next up

[Deep Dive into the Services - File Receiver Gateway](./ms-file-receiver-gateway.md)

BSD 3-Clause License: See [License](../LICENSE.md).
