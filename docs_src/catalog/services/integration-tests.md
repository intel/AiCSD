# Integration Test

Integration tests are automated to ensure that the containers work together properly. All these tests use a simulated
pipeline to demonstrate that data can be transferred throughout the system. Tests may be found in the `integration-tests`
directory. This directory also contains a Postman collection that may be imported to run additional manual integration tests.

## Run Integration Tests

### Prerequisites


Ensure that there is a clean environment.

1. Stop all services: 

   ``` bash 
   make down
   ```

2. To clean up the files from the directories on the Linux system, run:

    ```bash 
    make clean-files
    ```
   > **Warning**  
   > This is a **destructive** action and will delete any existing files.

3. To clean up the docker images, run:
    ``` bash 
    make clean-images
    ```


4. To remove any volumes from the system, run:

    ```bash
    make clean-volumes
    ```

   > **Warning**  
   > This is a **destructive** action and will clean up any unused docker volumes. This will delete the database and all its contents when run.

5. Use one of the command line options below to build: 

   General Build  

         make docker


   Fast Build

         make -j<num of threads> docker
    
    | Build Option  | When To Use |
    |:-----------------------|-------------------------|
    | General Build | The system resources are unknown. |
    | Fast Build  | The flag -j represents the number of jobs in a build. The optimal integer for the number of jobs varies, depending on the system resources (e.g., cores) and configuration. |

### Run Makefile Targets

There are several commands that may be used to run different configurations of the integration tests:

=== "Integration Test using Simulators"

| Run Option                                    | Description                                              |
|-----------------------------------------------|----------------------------------------------------------|
| ```make integration-test```              | Runs integration tests using the AiCSD simulators. |
| ```make integration-test-pipeline-sim``` | Runs the integration tests using the Pipeline Simulator. |
| ```make integration-test-retry```        | Runs the retry integration tests.                        |
    
> Note  
> The retry integration test target runs additional tests that include starting and stopping the containers to orchestrate more complex integration tests.

### Run Natively

1. Change to the integration test directory:

    ```bash
      cd integration-tests
    ```

2. Run the integration tests: 

    ```bash
      go test ./...
    ```

### Run within a Docker Container

1. Make the necessary directories:

    ```bash
       make files
    ```
2. Start the integration test container: 
   
    ```bash
    docker run -it -v $(pwd):/app  -v /var/run/docker.sock:/var/run/docker.sock -v ${HOME}:${HOME} --net=host aicsd/integration-tests:0.0.0-dev /bin/ash
    ```

3. Navigate to the `app` directory:

    ```bash
       cd app
    ```

4. Run the desired integration test target:

| Run Option                                    | Description                                              |
|-----------------------------------------------|----------------------------------------------------------|
| ```make integration-test```              | Runs integration tests using the AiCSD simulators. |
| ```make integration-test-pipeline-sim``` | Runs the integration tests using the Pipeline Simulator. |
| ```make integration-test-retry```        | Runs the retry integration tests.                        |

## Postman
The file `AiCSD.postman_collection.json` contains requests for manually testing using PostMan. The contents of this file will grow over time.

### Usage
1. From your **Workspace** in the PostMan application select **Import** and navigate to select the above collection file. 
2. Use the requests in the imported collection to create new Tasks, Get all Tasks, Get all Jobs, etc.

## Chaos Testing
Pumba is a command-line, chaos-testing tool for Docker.
It can be used to stress test containers, emulate network failures and modify the containers running.
Collecting metrics from different workloads within these test types assesses the robustness of the containers.

### Getting Started with Pumba
1. Clone the repository:
    ```bash
    git clone https://github.com/MUCZ/pumba
    ```
   
   > **Note**  
   > This is a specific fork of the repository `alexei-led/pumba.git` with compiler changes for `go-1.18`.

2. Change directories:
    ```bash
    cd pumba
    ```
3. Build the Pumba binary:
    ```bash 
    make
    ```
4. The binary may be found at `.bin/github.com/alexei-led`. To test the binary, print out the help resources:
    ```bash
    ./pumba --help
    ```

### Test with Pumba

Use Pumba to control containers:

- `stop`: stops specified containers
- `kill`: kills specified containers
- `rm`: removes specified containers
- `netem`: emulates network variances using `tc` traffic control tool

Use Pumba to control system resources:

- `stress`: stresses the specified containers using `stress-ng`
- `pause`: pauses all processes

### Sample Commands

**Network Emulation**: The following workload uses a docker image to host the traffic control commands.
The workload simulates a network delay of `30 ms` with a `5 ms` delay variation (jitter) that is normally distributed.
To find the containers to target, the following regex expression will select all containers that start with `edgex`.
    ```
    ./pumba netem --duration 1m --tc-image gaiadocker/iproute2 --interface eth1 delay --time 30 --jitter 5 --distribution normal "re2:^edgex"
    ```

**Stress Test**: The following will run a one minute stress test on the containers whose names start with `edgex`.

  ```
  ./pumba stress --stress-image alexeiled/stress-ng:latest-ubuntu -d 1m "re2:^edgex"
  
  ```

### Pumba Resources

[Pumba Repository](https://github.com/alexei-led/pumba)


