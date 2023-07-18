# Data Organizer Microservice

## Overview
The Data Organizer microservice routes calls from other microservices to the job repository. It acts as the gatekeeper and controller, determining if a call should create a new job in the job repository. Upon getting a call from the File Watcher, it queries the Task Launcher to determine if there are any tasks that match the input file. If no tasks match, the call errors out. If there is a matching task, the Data Organizer sends the job onwards to the File Sender OEM.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Job Repository](ms-job-repository.md)
- [Task Launcher](as-task-launcher.md)
- [File Sender OEM](ms-file-sender-oem.md)

### Attribute Parser

This service parses the filename of a job for attributes. To enable this process, specify a parsing schema in the `ms-data-organizer` configuration TOML under the `[AttributesParser]` section.

The Attribute requires three elements:

  - Name
  - ID
  - DataType

   ``` json
   [AttributeParser]
      [AttributeParser.Name]
      Id="Set search key letter/phrase"
      DataType="data type of the value (bool, int, or string)"
   ```

!!! Example
      In this example, the Names for each attribute are Flag, Number, and Operator. 
      
      The attributes have an associated ID and DateType.

      ``` json
      [AttributeParser]
        [AttributeParser.Flag]
        Id="f"
        DataType="bool"
        [AttributeParser.Number]
        Id="n"
        DataType="int"
        [AttributeParser.Operator]
        Id="op"
        DataType="string"
      ```

When the TOML is filled out correctly, the system automatically parses incoming filenames for their attributes using the given schema.
It searches for the `Id` in the file name and collects the corresponding data based on the `DataType`. It stores the data in a `map[string]string{AttributeName: data}`:

!!! example
      Input File: op-Bob-n007-f.tiff

      Resulting Attributes:
      ```json
      "Flag": "true",
      "Number": "7",
      "Operator": "Bob"
      ```

## Swagger Documentation

<swagger-ui src="./api-definitions/ms-data-organizer.yaml"/>

## Next up

[Deep Dive into the Services - Job Repository](ms-job-repository.md)

BSD 3-Clause License: See [License](../LICENSE.md).