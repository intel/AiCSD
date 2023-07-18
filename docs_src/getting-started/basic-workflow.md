# Basic Workflow
The following activity will require access to both systems (if using two). These steps assume that the systems have been set up following
the [System Setup Guide](./system-setup.md). 

!!! Note
    If running on a single system, all activities labeled **OEM** or **Gateway** will be completed on the same system.

**Objective:** Process an image file from an OEM microscope on the Gateway system.

1. On the **Gateway** system, create a task. A task is the method by which input images are matched to a processing pipeline.
    
    !!! Note
        This assumes that the pipeline(s) are running and configured. Either use the pipeline simulator or follow the [Image Classification Demo](../pipelines/bentoml/image-classification-demo.md) documentation to set up a ML pipeline.

    1. This can be done using **one** of the following methods:

        - **Web User Interface (UI)** - follow the instructions in the [Create Task](./ms-web-ui-guide-tasks.md#create-task) section.
        - **Postman** - use the integration-test collection [here](https://github.com/intel/AiCSD/blob/main/integration-tests/AiCSD.postman_collection.json)
        - **CURL command line tool** - developer exercise based on schema (e.g. Postman collection) or API definitions for the services

2. On the **OEM** system, drop a file to be processed.
    1. Once all the services are up and running, open a File Explorer window and enter `$HOME/data/oem-files/input` into the navigation bar. This is the default folder that the system will watch for files to be processed. If no GUI is available, use `cd $HOME/data/oem-files/input` in a Terminal to navigate to the input directory.
    2. Drop a file into the `$HOME/data/oem-files/input` folder using **one** of the methods below:

        - **Process an individual file** - Drag and drop an image file using the File Explorer. If no GUI is available, copy the file using `cp` in a Terminal.
        - **Process a directory of files** - Use the copy-files make target to copy all files in a directory to the input folder.

            ```bash
            make copy-files COPY_DIR=/path/to/dir
            ```

        !!! Note
            The default wait time between files being copied to the folder is 30 seconds. To change this, update the SLEEP_TIME variable.

                make copy-files COPY_DIR=/path/to/dir SLEEP_TIME=15
                

        !!! Note 
            Be sure that the Job Selector rule from creating a task is consistent with the file name. This is what will trigger an event to process the file.
     
    3. Check to see if there are any files in the `$HOME/data/oem-files/output` folder.

        !!! Note
            Depending on the chosen pipeline an output file may not be generated. For example, using the `results-only` pipeline on the pipeline simulator will not generate an output file.

3. On the **Gateway** system, check the status of the file and its associated job using one of the following methods:
       
    !!! Note
        The status will be Complete if the job has been successfully processed. Otherwise, the Owner field will show what component is processing the job.

      - **Web UI** - [View Jobs](./ms-web-ui-guide-jobs.md#job-management) on Web UI.
      - **Command Line** - follow the instructions below.
        1. Query for all jobs using `curl localhost:59784/api/v1/job | json_pp`. 
        2. Search the output for the desired input file name and grab the job id.
        3. Copy the job id and use it to query for details on the specified job `curl localhost:59784/api/v1/job/<job_id> | json_pp`.

    !!! Note
        To check the status of the pipeline processing the image, check the logs of the `edgex-pipeline-sim-1` container.

If the job did not complete, see [Troubleshooting](./troubleshooting.md).

## Next up
 
 [Learn more about the UI](./ms-web-ui-guide.md)


BSD 3-Clause License: See [License](../LICENSE.md).
