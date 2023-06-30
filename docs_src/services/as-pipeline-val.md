# Pipeline Validator

## Overview
This Application Service is a simulator to use as an alternative to the microservice infrastructure for initial integration.
The `StartPipeline` API will send an event to the pipeline to begin processing.
This simulator will serve the endpoints necessary for the Job Repository Update and the Pipeline Status calls.
The necessary integration points are outlined in the Creating Custom Pipelines section.

## Dependencies
This application service depends on the following services:

- [Consul from EdgeX](https://docs.edgexfoundry.org/2.3/security/Ch-Secure-Consul/)
- [Redis for EdgeX Message Bus](https://docs.edgexfoundry.org/2.3/microservices/general/messagebus/#redis-pubsub)

## Usage

1. Build the Pipeline Validator and the Pipeline Simulator (if using).
    ```bash
    make docker-pipeline-val docker-pipeline-sim
    ```
2. [Optional] If using a custom pipeline service, modify the `APPLICATIONSETTINGS_PIPELINEHOST` and `APPLICATIONSETTINGS_PIPELINEPORT` variables in the file `docker-compose-pipeline-val.yml`.

    !!! Note
        Refer to the table below to determine the values for `APPLICATIONSETTINGS_PIPELINEHOST` and `APPLICATIONSETTINGS_PIPELINEPORT` based on the desired Run Configuration. The default value is for the Pipeline Simulator.

3. To use the pipeline validator, use **one** of the following run options:

    | Run Configuration                                                | Run Command                                                                   | APPLICATIONSETTINGS_PIPELINEHOST | APPLICATIONSETTINGS_PIPELINEPORT
    |:------------------------------------------------------------------------------|:-----------------------|:-|:-|
    | Run with the pipeline simulator | `make run-simulators`                                                         | `pipeline-sim` | `59789` |
    | Run with a custom pipeline service | `make run-pipeline-val`                                                       | `<container_name>` | `<Docker_network_port>` |  

4. Open Postman and import the Postman collection from [here](https://github.com/intel/AiCSD/blob/main/as-pipeline-val/pipeline-val.postman_collection.json).
5. Verify that the Pipeline API works, by sending the `Get Pipelines` request. This shows all pipelines and their topics.
6. Create or copy a file for the pipeline to process in `$HOME/data/gateway-files/input`.

    !!! Note
        The location `$HOME/data/gateway-files` is volume mounted to `/tmp/files/` in the Docker container.

7. Modify the `Launch Pipeline` request so that the body contains the correct file name and the appropriate mqtt topic for the selected pipeline.
8. Click `Send` in Postman to send the request.
    
    !!! Note
        To monitor the status, check the pipeline container log files.

9. To check the status of the job, send the `Get Jobs` request. 

## Tear Down

1. Stop the services running.
    ```bash
    make down
    ```
2. Clean up the volumes. This is a destructive action, but will clear any configuration information that is in Consul.
    ```bash
    make clean-volumes
    ```
3. Clean up the files. This is also destructive as it will clear the input directory and the output directory.
    ```bash
    make clean-files
    ```

## Swagger Documentation

<swagger-ui src="./api-definitions/as-pipeline-val.yaml"/>

## Future Considerations
In the future, the Pipeline Validator may have a UI to display the job information and start the pipeline(s). 

INTEL CONFIDENTIAL: See [License](../LICENSE.md).