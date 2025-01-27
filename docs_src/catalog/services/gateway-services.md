# Gateway Services

Topics covered in this section:
-  [Job Repository](#job-repository)
-  [Web UI](#web-ui)
-  [File Receiver (Gateway)](#file-receiver-gateway)
-  [Task Launcher](#task-launcher)
-  [File Sender (Gateway)](#file-sender-gateway)

## Job Repository

### Overview
The Job Repository microservice provides a central location for managing information related to the processing of files/jobs.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis from EdgeX](https://docs.edgexfoundry.org/2.3/microservices/core/database/Ch-Redis/)

### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/ms-job-repository.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/ms-job-repository.yaml"/>

## Web UI

The AiCSD Web UI is an Angular interface that provides features such as:

* **Model** - Upload
* **Task** - Create, Update, Delete
* **Job** - Monitor, Filter, Verify
* **Dashboard** - View Analytics

### How do I build this thing?

This project is intended to be built and run using only `make` and `docker` without the need
for installing `npm`, `nodejs`, or `angular-cli`.

To skip the technical details, jump to **[Initial Setup](#initial-setup)**

If you want to know how the heck this thing works and what it is doing to your machine, 
read on to **[Build Ideology](#build-ideology)**

### Build Ideology

A lot of work has gone into making this friendlier to build and run. `Docker` and `make` are used to
allow the developer to work without ever installing `nodejs` or `angular-cli`.

A base nodejs docker image is built using your local user and group. When this image
is run, this entire repository is mounted under `/app` inside the container. This means whatever you do inside it
is also done locally as well. However, it also means that whatever you do locally is automatically
available to the docker side of things. This allows the developers to work completely
dockerized, completely native, or somewhere in-between.

Two docker files are provided, a development one and a production one. By default, the development one will be used. View the
**[Production](#production)** section to see how to build and run the production version.

`make` commands are used to abstract away the more complicated docker command line. 

`make debug` will launch an interactive build container to run any commands you would like (`npm`, `ng`, etc).

`make ng="..."` and `make npm="..."` allow you to run ng and npm commands within the confines of the
docker build container.

> **Note**  
> Following make commands with target as `serve`, `test`, and `debug` will work only on the gateway system where the Web UI microservice is built and run.

### Initial Setup

#### Create development docker image

```shell
make image
```

#### Install npm packages (node_modules)

```shell
make install
```

#### Build and serve development code

```shell
# Foreground
make serve
```

#### View Website

The website is available at <http://localhost:4200>.

Any changes to the code will be hot-reloaded by the angular development server (except in production mode).

##### Internationalization

The website supports the localization of many Job fields to Chinese.
To enable Chinese localization, 
set the `Accept-Language` header value to `zh`.
This may be set by adding an extension such as [Mod Header](https://chrome.google.com/webstore/detail/modheader-modify-http-hea/idgpnmonknjnojddfkpgkljpfnnfcklj) to the current web browser,
or via `Postman`, or `curl`.
Once the header configuration is set,
view the localized Job fields under the [Jobs tab](http://localhost:4200/jobs) of the UI.

#### Stop / Remove Containers

In order to stop the angular server which is in the foreground, use `Ctrl-C`. If that does not work, you can stop the running container with:

```shell
docker stop <container_name>
```

#### Other useful make commands

```shell
# Build and run unit tests
make test

# Open the unit test code coverage in your browser
make view-coverage

# Run linter
make lint

# Check for security issues with node packages
make audit

# Attempt to fix security issues with packages
make audit-fix

# Upgrade angular to the latest stable version
make update-angular

# Clean build cache, coverage info and dist files
make clean

# Clear out all node modules and build artifacts
make nuke

# Run an npm install
make install

# Run an npm install --force
make force-install

# Open an interactive build environment
# Desktop x11 environment is mapped and programs such as google-chrome can be ran in GUI mode
make debug

# Generate documentation
make docs

# Open documentation in web-browser
make view-docs

```

#### Advanced Make Commands

Open an interactive build environment to run any `npm`, `ng`, or other command within the confines
of the volume mounted development container

```shell
make debug
```

To run `npm` commands within the confines of the volume mounted development container, set the `npm` variable
to your `npm` argument list like so:

```shell
# This will run "npm audit --fix"
make npm="audit --fix"

# This will run "npm install [package-name] --save"
make npm="install moment --save"
```

To run `angular-cli` (`ng`) commands within the confines of the volume mounted development container, set the `ng` variable
to your `ng` argument list like so:

```shell
# This will run "ng generate component Test"
make ng="generate component Test"
```

To run any arbitrary command within the confines of the volume mounted development container, set the `exec`
variable to your command and argument list like so:

```shell
# This will run "node --version"
make exec="node --version"
```

### Code Generation

Run `make gen="component component-name"` to generate a new component.

You can generate the following schematics:

- app-shell
- application
- class
- component
- directive
- enum
- guard
- interceptor
- interface
- library
- module
- pipe
- resolver
- service
- service-worker
- web-worker

> **Note:**  
> Do not include the suffix (`Component`, `Service`, etc.). They are added for you_

### Testing

Run `make test` to execute the unit tests via [Karma](https://karma-runner.github.io).
You can view the Karma tests at the address `http://127.0.0.1:9876`. Changes will be hot-reloaded
and tests will be re-run.

Run `make view-coverage` to view static code coverage HTML output. Alternatively, open **/ms-web-ui/coverage/report-html/index.html** under project location, with your browser.

**Note:** In order for the code coverage to be accurate, you must first _refresh_ the Karma unit test service at <http://127.0.0.1:9876>.
It will then produce the HTML code coverage output for you to view in your browser as an HTML file.

### Native Tooling

In order to set up a native tooling environment without docker (for instance, use with an IDE), follow these steps to
download and install all dependencies.

#### Install NodeJS LTS

- _[Recommended]_ Install from [binary distributions](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions)
- Install via [snap](https://snapcraft.io/node)
  - _Note: Installations via snap will sometimes cause issues with installing global packages such as `angular-cli`_

#### Install Angular CLI

```shell
npm install -g @angular/cli
```

### Production

The production mode generates production built angular static html/js files able to be served via `nginx` or similar web server.

#### Build production files

```shell
make dist
```

## File Receiver (Gateway)

### Overview
The File Receiver Gateway microservice responds to TransmitJob and TransmitFile API endpoints. On startup, it queries for unprocessed job events. The File Receiver Gateway writes files sent to it from the File Sender OEM to the output directory specified in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-receiver-gateway/res/configuration.toml) file.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](#job-repository)
- [Task Launcher](#task-launcher)

### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/ms-file-receiver-gateway.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/ms-file-receiver-gateway.yaml"/>


## Task Launcher

### Overview
The Task Launcher microservice manages and launches tasks for jobs to be executed on the Pipeline Simulator, Intel® Geti™ platform pipelines, or BentoML pipelines.

### Dependencies
This application service is dependent on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis from EdgeX](https://docs.edgexfoundry.org/2.3/microservices/core/database/Ch-Redis/)
- [Redis for EdgeX MessageBus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [Job Repository](#job-repository)
- [File Sender (Gateway)](#file-sender-gateway)

> **Note**  
> The same Redis implementation is used both for the database and the Publish/Subscribe Message Broker needed for the EdgeX Message Bus.

### Configuration
Change task launcher configurations in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/as-task-launcher/res/configuration.toml) file. Configuration options can also be changed when the service is running by using [Consul](http://localhost:8500/ui/dc1/kv/edgex/appservices/2.0/as-task-launcher/ApplicationSettings/).

> **Note**  
> For changes to take effect, the service must be restarted. If changes are made to the configuration.toml file, the service must be stopped, rebuilt, and started again.

- **RetryWindow:** Determines how often a job should be resent to the pipeline for processing
- **DeviceProfileName:** Indicates the device profile information for the pipeline to consume
- **DeviceName:** Indicates the device name for the pipeline to consume


### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/as-task-launcher.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/as-task-launcher.yaml"/>


## File Sender (Gateway)

### Overview
The File Sender Gateway microservice responds to the DataToHandle, TransmitFile, ArchiveFile, and RejectFile API endpoints.
It sends the `Job` (received in DataToHandle) to the File Receiver OEM via the EdgeX Message Bus
and accepts requests from the File Receiver OEM to pull the files once it receives the `Job`.
After the output file(s) are successfully written to the OEM system, it is archived on the Gateway.
If jobs are rejected in the Web UI the File Sender Gateway copies the archived image to `$HOME/data/gateway-files/reject`.

!!! Note
    During the archival process, if the file is not a web viewable type (`.png, .jp(e)g, or .gif`), then an image conversion process is executed and a `.jpeg` image is created for use in the Web UI.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [Job Repository](#job-repository)


### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/as-file-sender-gateway.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/as-file-sender-gateway.yaml"/>





