# Setup Prerequisites on OEM and Gateway Systems

## Setup WSL2 for Windows OEM system

1. Install WSL using the Windows store
1. In Powershell, set WSL to version 2
    ```powershell
    wsl --set-default-version 2
    ```
1. Install the following tools in WSL2 following the steps under [Install Prereqs](#install-prerequisites)
    * git
    * make
    * curl
    * Docker Engine
1. Edit `/etc/wsl.conf`
    * Enable systemd
    ```bash
    [boot] systemd = true
    ```
    * Disable generate hosts
    ```bash
    [network] generateHosts = false
    ```

1. To get the software on the OEM device, clone the AiCSD code repository using **one** of the options below using WSL terminal:

    === "HTTP"
         ```
         git clone https://github.com/intel/AiCSD.git
         ```
    === "SSH"
         ```
         git clone git@github.com:intel/AiCSD.git
         ```

	!!! Note
        To update to a specific version of the software after cloning, use:

		``` bash
		git checkout <version-tag>
		```

1. Setup WSL2 Port Forwarding within OEM device
    * Using Admin Powershell on OEM Device, run the `WSL_Port_Setup_Script.ps1` powershell script.

    !!! Note
        To remove the port forwarding rules, run the `WSL_Port_Removal_Script.ps1` powershell script.

## Setup Ubuntu for Linux OEM or Gateway System

The only required setup for these systems is to install [Ubuntu 20.04](https://releases.ubuntu.com/focal/).

## Install Prerequisites

Install make, git, curl, Docker, and Docker Compose.

1. Update repositories: 
   ```bash
   sudo apt-get update
   ```
1. Install make, git, and curl:
   ```bash
   sudo apt-get install -y make git curl
   ```
1. [Install Docker Engine.](https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository)

    !!! Note
        Only follow the steps for *Install Using the Repository* in the docker setup guide linked above. This will set up the repository and install the Docker Engine and Docker Compose, which are necessary for setting up the microservices.

1. Follow these post-installation steps to [Manage Docker as a non-root user](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user).

1. Ensure access and configurations to be able to clone the AiCSD code repository using either SSH or HTTPS. For SSH, refer to [Generating a new SSH key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent) and add the public key to GitHub profile account settings. For HTTPS, refer to [Creating a Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) for authentication purposes. Treat the key as a password.


## Download Repository

1. To get the software, clone the AiCSD code repository using **one** of the options below:

    === "HTTP"
         ```
         git clone https://github.com/intel/AiCSD.git
         ```
    === "SSH"
         ```
         git clone git@github.com:intel/AiCSD.git
         ```

	!!! Note
        To update to a specific version of the software after cloning, use:

		``` bash
		git checkout <version-tag>
		```

1. Navigate to the working directory:

    ```bash
    cd AiCSD
    ```

## Next up

[Gateway Installation > Configure](choose-config.md)

BSD 3-Clause License: See [License](../LICENSE.md).