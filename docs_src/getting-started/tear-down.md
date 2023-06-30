# Stop Services and Clean Up
The section details how to stop all the software services and clean up the environments. It can also be useful when [Troubleshooting](troubleshooting.md).

## Stop Services on Gateway (or single system)
To stop the services, run

``` bash
make down
```

This will bring down all services that are running.

## Stop Services on OEM
To stop the services, run

``` bash
make down-oem
```

This will bring down all services that are running.

## Clean Up for Any System
1. To clean up the docker images, run:

    ``` bash 
    make clean-images
    ```
2. To clean up the files from the directories on the Linux system, run:

    ```bash 
     make clean-files
    ```

    !!! Warning
        This is a **destructive** action that will delete input and output files in `oem-files` and `gateway-files` folders under `$HOME/data/` as well as generated secrets files under `/tmp/edgex/secrets`.

3. To remove any volumes from the system, run:

    ```bash
    make clean-volumes
    ```

    !!! Warning
        This is a **destructive** action and will clean up any unused docker volumes. This will delete the database and all its contents when run.
4. To remove any generated ssh keys from the system, run:

 ```bash
 make clean-keys
 ```

!!! Warning
This is a **destructive** action and will clean up any generated ssh keys. This will mean that the keys need to be regenerated on the Gateway and copied onto the OEM system.
