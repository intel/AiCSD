# OEM Services

Topics covered in this section:
-  [File Watcher](#file-watcher)
-  [Data Organizer](#data-organizer)
-  [File Sender (OEM)](#file-sender-oem)
-  [File Receiver (OEM)](#file-receiver-oem)

## File Watcher

### Overview
The File Watcher microservice watches specified folders for new files. Upon startup, it queries selected folder(s) on a local system for unprocessed files. When it identifies a new file(s), it makes a REST call to the data organizer with the job containing the new file information. 

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Data Organizer](#data-organizer)

### Configuration
The File Watcher microservice has a number of configurations that can be changed in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-watcher/res/configuration.toml) file. For most of these changes to be reflected, the File Watcher container must be restarted. However, the settings listed below can be manipulated while the service is running by using Consul `Key/Values/edgex/appservices/2.0/ms-file-watcher/`:

- **WatchSubfolders:** Alerts the microservice to search through nested file structures.
- **FileExclusionList:** Blocks certain files from being processed with a comma-separated list of substrings. 
- **LogLevel:** Determines verbosity of logging output.

> **Example**  
>    ** FileExclusionList Substrings **
>
>    `FileExclusionList="test, image-4k"`
>
>    These files **would not** be processed as their names contain whole substrings from the file exclusion list:
>
>    `test-image.tiff` and `image-4k.tiff` 
>
>    This file **would be** processed:
>    
>    `image.tiff`


### Usage
This Device Service runs standalone or with EdgeX services. It must have communication via REST API to the
data organizer in order for data to be processed.


## Data Organizer

### Overview
The Data Organizer microservice routes calls from other microservices to the job repository. It acts as the gatekeeper and controller, determining if a call should create a new job in the job repository. Upon getting a call from the File Watcher, it queries the Task Launcher to determine if there are any tasks that match the input file. If no tasks match, the call errors out. If there is a matching task, the Data Organizer sends the job onwards to the File Sender OEM.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](./gateway-services.md#job-repository)
- [Task Launcher](./gateway-services.md#task-launcher)
- [File Sender (OEM)](#file-sender-oem)

#### Attribute Parser

This service parses the filename of a job for attributes. To enable this process, specify a parsing schema in the `ms-data-organizer` configuration TOML under the `[AttributesParser]` section.

The Attribute requires three elements:

  - Name
  - ID
  - DataType

         [AttributeParser]
         [AttributeParser.Name]
         Id="Set search key letter/phrase"
         DataType="data type of the value (bool, int, or string)"

> **Example**  
> In this example, the Names for each attribute are Flag, Number, and Operator. 
>      
>      The attributes have an associated ID and DateType.
>
>      [AttributeParser]
>        [AttributeParser.Flag]
>        Id="f"
>        DataType="bool"
>        [AttributeParser.Number]
>        Id="n"
>        DataType="int"
>        [AttributeParser.Operator]
>        Id="op"
>        DataType="string"


When the TOML is filled out correctly, the system automatically parses incoming filenames for their attributes using the given schema.
It searches for the `Id` in the file name and collects the corresponding data based on the `DataType`. It stores the data in a `map[string]string{AttributeName: data}`:

> **Example**  
>      Input File: op-Bob-n007-f.tiff
>
>      Resulting Attributes:
>      "Flag": "true",
>      "Number": "7",
>      "Operator": "Bob"


### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/ms-data-organizer.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/ms-data-organizer.yaml"/>


## File Sender (OEM)

### Overview
The File Sender OEM microservice listens for events and sends files from those events. On startup, it queries for
unprocessed job events. The File Sender OEM sends files received from the data organizer to the 
File Receiver Gateway. The configuration information is set in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-sender-oem/res/configuration.toml) file.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](./gateway-services.md#job-repository)
- [File Receiver (Gateway)](./gateway-services.md#file-receiver-gateway)

### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/ms-file-sender-oem.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/ms-file-sender-oem.yaml"/>


## File Receiver (OEM)

### Overview
The File Receiver OEM microservice receives the `Job(s)` object via the EdgeX Message Bus.
It pulls the `Job` file(s) from the File Sender Gateway via the TransmitFile API endpoint.
After a file is successfully written to the OEM system, it is archived on the Gateway.

### Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)
- [File Sender (Gateway)](./gateway-services.md#file-sender-gateway)
- [Job Repository](./gateway-services.md#job-repository)

### Swagger Documentation

You can find the API definition in `AiCSD/docs_src/services/api-definitions/as-file-receiver-oem.yaml`, and use Swagger API* software or the Postman* software to render the API.

<swagger-ui src="./api-definitions/as-file-receiver-oem.yaml"/>



