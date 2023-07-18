# File Watcher Microservice

## Overview
The File Watcher microservice watches specified folders for new files. Upon startup, it queries selected folder(s) on a local system for unprocessed files. When it identifies a new file(s), it makes a REST call to the data organizer with the job containing the new file information. 

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Data Organizer](ms-data-organizer.md)

## Configuration
The File Watcher microservice has a number of configurations that can be changed in the [configuration.toml](https://github.com/intel/AiCSD/blob/main/ms-file-watcher/res/configuration.toml) file. For most of these changes to be reflected, the File Watcher container must be restarted. However, the settings listed below can be manipulated while the service is running by using Consul `Key/Values/edgex/appservices/2.0/ms-file-watcher/`:

- **WatchSubfolders:** Alerts the microservice to search through nested file structures.
- **FileExclusionList:** Blocks certain files from being processed with a comma-separated list of substrings. 
- **LogLevel:** Determines verbosity of logging output.

!!! Example 
    ** FileExclusionList Substrings **

    `FileExclusionList="test, image-4k"`

    These files **would not** be processed as their names contain whole substrings from the file exclusion list:

    `test-image.tiff` and `image-4k.tiff` 

    This file **would be** processed:
    
    `image.tiff`


## Usage
This Device Service runs standalone or with EdgeX services. It must have communication via REST API to the
data organizer in order for data to be processed.


## Next up

[Deep Dive into the Services - Data Organizer](ms-data-organizer.md)

BSD-3 License: See [License](../LICENSE.md).
