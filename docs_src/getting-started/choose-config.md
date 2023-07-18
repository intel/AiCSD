# Configure

## Generate Keys (Optional)

This is only **required in a two-system** setup. In order to use the SSH tunnel between the systems, it is necessary to generate SSH keys. 
Only the public key will be shared to the OEM system. Each keypair should be unique to its deployment for security purposes.

1. Generate ssh keys on the Gateway System: `make generate-ssh-tunnel-keys`
2. Copy generated public key to the OEM system filling in the appropriate username for `<oem-user>`, system name for `<OEM-System>`, and path to the repository `/path/to`.
    ```bash
    $ scp -r edgex-res/remote/sshd-remote/authorized_keys <oem-user>@<OEM-System>:/path/to/applications.retail.kaskey-park.kaskey-park/edgex-res/remote/sshd-remote/authorized_keys
    ```
   
    !!! Note
        If it is not possible to use `scp` to move the file, use a USB flash drive to move the file `edgex-res/remote/sshd-remote/authorized_keys` on the gateway to `/path/to/applications.retail.kaskey-park.kaskey-park/edgex-res/remote/sshd-remote/authorized_keys` on the OEM.

## Modify Firewall Rules (Optional)

This is only **required in a two-system setup with a Windows OEM** system. 
Depending on the network setup, it may be necessary to create an inbound rule in order to allow traffic in for SCP on port 2222 and for the port forwarding on port 2223.
Creating inbound/outbound rules on Windows can be done following the instructions [here](https://learn.microsoft.com/en-us/windows/security/operating-system-security/network-security/windows-firewall/best-practices-configuring).
To turn off the Windows Defender Firewall, follow the steps [here](https://support.microsoft.com/en-us/windows/turn-microsoft-defender-firewall-on-or-off-ec0844f7-aebd-0583-67fe-601ecf5d774f).

## Configure Specific Services

If a custom configuration option is not needed, return to the main installation flow, [Build and Deploy](system-setup.md). 

The table below provides links to configuration options along with the computer it should run on (if using a two system setup). 

| Component                             | Description           | Modify Configuration To  | 
|:--------------------------------------|-----------------------|----------------------------|
| [File Watcher](../services/ms-file-watcher.md#configuration) | The File Watcher component monitors selected folder(s) for new files to process.  | Exercise greater control over the File Watcher component. (e.g., Refinement of file filtering)   | 
| [Data Organizer > Attribute Parser](../services/ms-data-organizer.md#attribute-parser) | The Data Organizer helps route calls from other microservices to the job repository. The attribute parser of the data organizer can parse the file name of a job for attributes.  | Customize file name information gathering.   |
| [Task Launcher > RetryWindow ](../services/as-task-launcher.md#configuration) | The Task Manager manages and launches tasks for jobs to be executed on the Pipeline Simulator, Geti pipelines, or BentoML pipelines  | Set the frequency for resubmitting jobs to a pipeline.    |

## Next up

[Build and Deploy](system-setup.md)

BSD 3-Clause License: See [License](../LICENSE.md).