# File Receiver OEM Application Service

## Overview
The File Receiver OEM microservice receives the `Job(s)` object via the EdgeX Message Bus.
It pulls the `Job` file(s) from the File Sender Gateway via the TransmitFile API endpoint.
After a file is successfully written to the OEM system, it is archived on the Gateway.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [File Sender Gateway](./as-file-sender-gateway.md)
- [Job Repository](./ms-job-repository.md)

## Swagger Documentation

<swagger-ui src="./api-definitions/as-file-receiver-oem.yaml"/>

## Next up

[Deep Dive into the Services - Web User Interface](./ms-web-ui.md)

BSD-3 License: See [License](../LICENSE.md).
