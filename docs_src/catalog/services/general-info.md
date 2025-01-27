# Services Information
This section provides general information for the overall collection of services at a system-level. Select the 
hyperlinked name of the microservice in the **[Port Configurations](#port-configurations)** table below for service-level information.

## Image Processing

Images are processed and tracked using jobs. The job tracks the movement of the file, the status, and any results or outputs 
from the pipeline. To process a job, there are tasks that help match information about a job to the appropriate pipeline 
to run. The animation below demonstrates how a file gets automatically processed.

![FileMovement](../images/aicsd-arch-animation.gif)

1. The Input Device/Imager writes the file to the OEM file system in a directory that is watched by the File Watcher. When 
the File Watcher detects the file, it sends the job (JSON struct of particular fields) to the Data Organizer via HTTP Request.

2. The Data Organizer sends the job to the Job Repository to create the job in the Redis Database. The job information is then 
sent to the Task Launcher to determine if there is a task that matches the job. If there is, the job may proceed to the 
File Sender (OEM). 

3. The File Sender (OEM) is responsible for sending both the job and the file to the File Receiver (Gateway). Once the File
Receiver (Gateway) has written the file to the Gateway file system, the job may then be sent on to the Task Launcher. 

4. The Task Launcher verifies that there is a matching task for the job before sending it to the appropriate pipeline using
the EdgeX Message Bus (via Redis). The ML pipeline subscribes to the appropriate topic and processes the file in its pipeline.
The output file (if there is one) is written to the file system and the job is sent back to the Task Launcher. 

5. The Task Launcher then decides if there is an output file or if there are just results. In the case of only results and
no output file, the Task Launcher marks the job as complete. If there is an output file, the Task Launcher sends the job
onwards to the File Sender (Gateway). 

6. The File Sender (Gateway) publishes the job information to the EdgeX Message Bus via Redis for the File Receiver (OEM) to subscribe
and pull. The File Receiver (OEM) sends an HTTP request to the File Sender (Gateway) for the output file(s). 
The file(s) are sent as part of the response and the File Receiver (OEM) writes the output file(s) to the file system.


## Port Configurations

| Microservice                                         |  Port |
|:-----------------------------------------------------|------:|
| [file-watcher](./oem-services.md#file-watcher)                   | 59780 |
| [data-organizer](./oem-services.md#data-organizer)               | 59781 |
| [file-sender-oem](./oem-services.md#file-sender-oem)             | 59782 |
| [file-receiver-gateway](./gateway-services.md#file-receiver-gateway) | 59783 |
| [job-repository](./gateway-services.md#job-repository)               | 59784 |
| [task-launcher](./gateway-services.md#task-launcher)                 | 59785 |
| [file-sender-gateway](./gateway-services.md#file-sender-gateway)     | 59786 |
| [file-receiver-oem](./oem-services.md#file-receiver-oem)         | 59787 |
| [web-ui](./gateway-services.md#web-ui)                               |  4200 |
| [pipeline-sim](./pipeline-services.md#pipeline-simulator)                   | 10107 |
| [pipeline-val](./pipeline-services.md#pipeline-validator)                   | 59788 |

| Dependencies                                             |                     Port |
|:---------------------------------------------------------|-------------------------:|
| Consul                                                   |                     8500 |
| Redis                                                    |                     6379 |
| App MQTT Export                                          |                    59703 |
| External MQTT Broker                                     |                     1883 |
 | InfluxDB                                                 |                     8086 |
 | Grafana                                                  |                     3001 |
 | Kibana                                                   |                     5601 |

| EdgeX Security Services        | Port               |             Which System | 
|:-------------------------------|--------------------|-------------------------:|
| Kong                           | 8000, 8443         | Gateway or Single System |
| Kong DB                        | 5432               | Gateway or Single System |
| Security Proxy Setup           | None               | Gateway or Single System |
| Security SecretStore Setup     | None               | Gateway or Single System |
| Security Bootstrapper          | None               | Gateway or Single System |
| Vault                          | 8200               | Gateway or Single System |
| Security Proxy Setup           | 59780-59782, 59787 |             Gateway Only |
| Security Spiffe Token Provider | 59841              |             Gateway Only |
| Security Spire Agent           | None               |             Gateway Only |
| Security Spire Config          | None               |             Gateway Only |
| Security Spire Server          | 59840              |             Gateway Only |
| SSHd Remote                    | 2223               |                 OEM Only |
| Remote Spire Agent             | None               |                 OEM Only |

> **Note**  
> For more information on the Security Services for EdgeX, refer to the [EdgeX Security Documentation](https://docs.edgexfoundry.org/2.3/security/Ch-Security/).
> The two system secure port forwarding is implemented based on the [EdgeX example](https://docs.edgexfoundry.org/2.3/security/Ch-RemoteDeviceServices/) for remote device services in secure mode.

## Build Options

The top level Makefile contains the following make targets for building the service binaries and docker images.

| Option                       | Description                           |
|:-----------------------------|:--------------------------------------|
| `make tidy`                  | Runs `go mod tidy` to ensure the go.sum file is up to date. Only needed once if `build` fails due to a go.sum issue. |
| `make build`                 | Builds all the AiCSD microservice binaries. |
| `make <service-name>`        | Builds the specified microservice binary. Microservice names listed in the table above are used as the make targets. The `<service-name>` is any name listed in the [Microservices](#port-configurations) table. |
| `make docker`                | Builds all AiCSD microservice docker images. Adding the option `-j<threads>` will tell make how many commands to run in parallel, where `<threads>` is the desired number of threads. |
| `make docker-build-gateway`  | Builds the AiCSD Gateway specific microservice docker images. Adding the option `-j<threads>` will tell make how many commands to run in parallel, where `<threads>` is the desired number of threads. |
| `make docker-build-oem`      | Builds the AiCSD OEM specific microservice docker images. Adding the option `-j<threads>` will tell make how many commands to run in parallel, where `<threads>` is the desired number of threads. |
| `make docker-<service-name>` |                                                                                       Builds the specified microservice docker image. The `<service-name>` is any name listed in the [Microservices](#port-configurations) table. | 
| `make files`                 | Creates local folders for the OEM and Gateway files. Dependency of the `run*` targets below. |

## Run Options

The top level Makefile contains the following make targets for running the microservices in docker.

> **Note**  
> The AiCSD docker images are not pushed to any Docker Registry. They must be built locally prior to using the target(s) below that depend on those docker images.

| Option                                           | Description                                         |
|:-------------------------------------------------|:----------------------------------------------------|
| `make run-gateway GATEWAY_IP_ADDR=192.168.XX.XX` | Runs all the Gateway targeted service containers including EdgeX and AiCSD services. Intended for use with a separate OEM system and a custom pipeline configuration. |
| `make run-gateway-sim`                           | Runs the Gateway services with a pipeline simulator. Used for integration testing or development with a separate OEM system. |
| `make run-gateway-geti`                          | Runs the Gateway services with Intel® Geti™ platform for pipeline creation. Requires a separate OEM system. |
| `make run-ovms`                                  | Runs the AiCSD Gateway services with a pipeline simulator. Used for integration testing or development with a separate OEM system. |
| `make run GATEWAY_IP_ADDR=192.168.XX.XX`              |  Runs the OEM and Gateway targeted service containers including EdgeX and AiCSD services on a single system. Intended for use with a separate OEM system and a custom pipeline configuration.  |
| `make run-sim`                                   | Runs the OEM and Gateway services with a pipeline simulator. Used for integration testing or development.  | 
| `make run-geti`                                  | Runs the OEM and Gateway services with Intel® Geti™ platform for pipeline creation. |
| `make run-oem`                                   | Runs the OEM services. Used for integration testing or development with a separate Gateway system.  |

!!! Note
    Appending the gateway IP Address `GATEWAY_IP_ADDR=192.168.XX.XX` will allow the web UI to be accessed from a remote system. To get the Gateway IP Address run `hostname -I` in a terminal on the Gateway system.

## Clean-Up Options

The following options are available for tearing down and cleaning up the solution.

| Option                | Description                                                                                                                                                                                                            | 
|:----------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `make down`           | Stops all containers no matter which target was used to start them.                                                                                                                                                    |
| `make down-oem`       | Stops all OEM containers.                                                                                                                                                                                              
| `make down-clean`     | Stops all containers and removes all the volumes. This will result in data loss.                                                                                                                                       |
| `make clean-files`    | Removes the local folders for the OEM and Gateway files. Removes EdgeX Secrets with sudo permissions. This will result in data loss and any configured pipelines will be lost.                                         |
| `make clean-volumes`  | Removes all unused Docker volumes. This will result in data loss. This command will work successfully for Docker version >= 23.0. However for Docker version < 23.0 use the following command: `docker volume --prune` | 
| `make clean-images`   | Removes all the locally built AiCSD Docker images.                                                                                                                                                                     | 
| `make clean-builders` | Removes all the "builder" images left over from the docker build process.                                                                                                                                              |
| `make clean-keys`     | Removes all the ssh keys from their directories for either the Gateway or OEM system.                                                                                                                                  |

## Portainer

Portainer is a service with a web UI that can be used for container management.

| Option                |                                                                                      Description |
|:----------------------|:-------------------------------------------------------------------------------------------------|
| `make run-portainer`  | Runs the Portainer container management application independent of the AiCSD services. |
| `make down-portainer` |                                                                   Stops the Portainer container. |

## Validation

The following validation test options are used to run unit and integration tests. For test reports, `go-test-report` is used to write test output to html files.

| Option                                                                                | Description                                                                                                                                                                                                                                      |
|:--------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `make test`                                                                           | Runs the unit tests locally.                                                                                                                                                                                                                     |
| `make integration-test`                                                               | Runs the basic integration tests.                                                                                                                                                                                                                |
| `make integration-test-retry`                                                         | Runs all integration tests including the retry test cases that will test the starting and stopping of the services. Improper synchronization of the containers stopping and starting can cause these tests to fail.                              |
| `make unit-test-report`                                                               | Runs the unit tests locally and outputs the results and coverage to _unit-test-report.html_ and _unit-test-cov-report.html_ respectively.                                                                                                        |
| `make integration-test-report`                                                        | Runs all integration tests including the retry test cases that will test the starting and stopping of the services. The results are output to _integration-test-report.html_.                                                                    |
| `make test-report`                                                                    | Runs the unit tests and integration test targets. It will generate output files _unit-test-report.html_ for the unit tests, _unit-test-cov-report.html_ for the unit test coverage and _integration-test-report.html_ for the integration tests. |
| `make copy-files COPY_DIR=/path/to/input-images/to-copy SLEEP_TIME=<time in seconds>` | Used for manual testing on the OEM/single system setup for copying files from a specified COPY_DIR to the appropriate location while waiting the specified SLEEP_TIME (default 30s).                                                             |

### Run Integration Tests in a Docker Container

1. Build the docker image:

    ```bash
    make docker-integration-test
    ```
   
2. Run the container:

    ```bash
    docker run --net host -v /var/run/docker.sock:/var/run/docker.sock -it --entrypoint="/bin/sh" aicsd/integration-test:0.0.0-dev
    ```

3. Once inside the shell, execute:

    ```bash
    make integration-test
    ```

## Services Fault Tolerance 

The services use a Go module called `wait-for-it` to wait on the availability of a TCP host and port.
The `wait-for-it` Go module is added to the microservices so that the services may wait for their dependencies to be up and ready as expected. It currently has a 15-second timeout and provides feedback as the dependent services become ready. If for some reason a service never becomes available, there is a one-minute maximum timeout, after which an error will be logged that a dependent service never became available.

## Documentation Using GitHub Pages
This repository leverages a GitHub Pages approach to represent markdown contents as navigable html web pages. To build and view the documentation locally, use:

   ```bash
   make serve-docs
   ```

> **Note**  
> Open a browser to view the contents: [localhost:8008](http://localhost:8008/)


```{toctree}
:maxdepth: 5
:hidden:
oem-services.md
gateway-services.md
pipeline-services.md
integration-tests.md
```