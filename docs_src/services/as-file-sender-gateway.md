# File Sender Gateway Application Service

## Overview
The File Sender Gateway microservice responds to the DataToHandle, TransmitFile, ArchiveFile, and RejectFile API endpoints.
It sends the `Job` (received in DataToHandle) to the File Receiver OEM via the EdgeX Message Bus
and accepts requests from the File Receiver OEM to pull the files once it receives the `Job`.
After the output file(s) are successfully written to the OEM system, it is archived on the Gateway.
If jobs are rejected in the Web-UI the File Sender Gateway copies the archived image to `$HOME/data/gateway-files/reject`.

!!! Note
    During the archival process, if the file is not a web viewable type (`.png, .jp(e)g, or .gif`), then an image conversion process is executed and a `.jpeg` image is created for use in the Web-UI.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [Job Repository](./ms-job-repository.md)


## Swagger Documentation

<swagger-ui src="./api-definitions/as-file-sender-gateway.yaml"/>

## Next up

[Deep Dive into the Services - File Receiver OEM](./as-file-receiver-oem.md)

BSD-3 License: See [License](../LICENSE.md).
