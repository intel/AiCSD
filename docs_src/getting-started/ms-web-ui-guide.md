# Web User Guide
This guide provides a description of the UI and instructions for managing tasks and jobs.

## Prerequisites

For successful operation, ensure that all microservices are running. If Portainer is up, you can verify services <a href="http://127.0.0.1:9000" target="_blank">here</a>.

!!! Note
    If running distributed services (i.e., 2-system configuration of Linux Edgebox and WSL2/Linux OEM system), 

    then use the IP address rather than localhost or the URL http://127.0.0.1.


<figure class="figure-image">
<img src="..\images\Portainer.jpg" alt="Figure 1: Portainer Container List">
<figcaption>Figure 1: Portainer Container List</figcaption>
</figure>


## Interface Basics

The landing page shown in Figure 2 opens to the **Task Management** tab as the default. The UI enables management of: 

* **Tasks** define how input files are matched to a processing pipeline.

* **Jobs** define the input file and track the status of the input file as it is processed. Jobs also track the results
  and/or output files for the given input file. 

* **Analytics** are gathered and displayed using [Grafana](./../monitoring/grafana.md).

* **Models** define a processing pipeline.

Figure 2 presents:

1. **Theme Toggle:** control the visual theme

2. **Create/Modify Tasks:** choose this option to view and manage Tasks

3. **View Jobs:** choose this option to view and manage Jobs

4. **Dashboards:** choose this option to view and manage analytics

5. **Upload Models:** choose this option to upload a model for use by a Task

<figure class="figure-image">
<img src="..\images\LandingPage.jpg" alt="Figure 2: Landing Page">
<figcaption>Figure 2: Landing Page</figcaption>
</figure>

### Access
To access the Web UI, click <a href="http://127.0.0.1:4200/" target="_blank">here</a>.

!!! Note
    If running distributed services (i.e., 2-system configuration of Linux Edgebox and WSL2/Linux OEM system), use the IP address rather than localhost or the URL http://127.0.0.1.



### Set Theme

There are two themes, illustrated with the Figures below:

<figure class="figure-image">
<img src="..\images\LightTheme.jpg" alt="Figure 3: Light Theme">
<figcaption>Figure 3: Light Theme</figcaption>
</figure>

<figure class="figure-image">
<img src="..\images\DarkTheme.jpg" alt="Figure 4: Dark Theme">
<figcaption>Figure 4: Dark Theme</figcaption>
</figure>

Toggle between the two themes by clicking the sun-moon button as shown in Figures 3 and 4.

## Next up

* Learn more about [Tasks](./ms-web-ui-guide-tasks.md)
* Learn more about [Jobs](./ms-web-ui-guide-jobs.md)
