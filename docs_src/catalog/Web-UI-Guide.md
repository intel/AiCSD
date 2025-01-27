# Web UI Features

## Prerequisites

You have completed the following:
-  Installed the software package and validated the installation according to the steps in the [Get Started Guide](./Get-Started-Guide.md).
-  Verified that all microservices are running. If Portainer is up, you can verify services <a href="http://127.0.0.1:9000" target="_blank">here</a>. An example is shown in Figure 1.

> **Note**  
> If running distributed services (i.e., two-system configuration of Linux and WSL2/Linux OEM system), then use the IP address rather than localhost or the URL http://127.0.0.1.

![Portainer Container List](./images/Portainer.jpg)

Figure 1: Portainer Container List

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

![Landing Page](./images/LandingPage.jpg)

Figure 2: Landing Page

### Access
To access the Web UI, click <a href="http://127.0.0.1:4200/" target="_blank">here</a>.

> **Note**  
> If running distributed services (i.e., 2-system configuration of Linux and WSL2/Linux OEM system), use the IP address rather than localhost or the URL http://127.0.0.1.

## Task Management

Topics covered under task management:
-  [Display All Tasks](#display-all-tasks)
-  [Create Task](#create-task)
-  [Update Task](#update-task)
-  [Delete Task](#delete-task)

### Display All Tasks

The default landing page is **Task Management**. Clicking the **Create/Modify Tasks** at the top-left of the navigation bar also displays the **Task Management** page. This page will display all the tasks currently available in the system.

   ![Display All Tasks](./images/DisplayAllTasks.jpg)

   Figure 3: Display All Tasks

The UI, shown in Figure 1, offers different ways to look at the current tasks:

1. **Filter:** enter text in the text box to filter tasks.

2. **Sort:** click on individual columns to sort tasks either in ascending or descending order.

3. **View pages:** use pagination, provided at the bottom of the page, to display a set number of tasks on the page.

### Create Task

To create a task: 

1. From Task Management page, select **Add Task**. 

      ![Create Task](./images/CreateTask.jpg)

      Figure 4: Create Task

2. Complete the following fields with the task details:
        
      > **Note**  
      > Regular expressions are not supported.  
      > For example, in **Job Selector** below, use of **contains** with "\*.tiff" will filter for tasks with the literal substring "\*.tiff".

    -  **Description**: enter Task Description
  
    -  **Pipeline**: click to select the Model Execution Pipeline from the dropdown
  
    -  **Job Selector**: click to select between the two checkboxes -
  
        --  **matches**: filter jobs based on filename that exactly match “filename” field value from below
    
        --  **contains**: filter jobs based on filename that contain “filename” field value from below
    
    -  **Filename**: enter input image filename   

    -  **Model Parameters**: enter parameter details for the model, it should adhere to the following json format, including quotes – {“parameter” : ”value”}

      ![Task Form](./images/FillFormTask.jpg)

      Figure 5: Task Form

3. Click **Save**.

### Update Task

1. Click on **Update** for the task that needs to be modified:

      ![Update Task](./images/UpdateTask.jpg)

      Figure 6: Update Task

2. Modify the appropriate field (i.e., **Description** modified)

      ![Modify Field Task](./images/ModifyFieldTask.jpg)

      Figure 7: Modify Field Task

3. Click **Save**.
4. Confirm that the changes are saved as shown below:

      ![Updated Task](./images/UpdatedTask.jpg)

      Figure 8: Updated Task


### Delete Task

1. Select one or more checkboxes next to the entry/entries to delete.
2. Click **Delete Selected**. 
3. Confirm the deletion:

      ![Task Deletion](./images/DeleteTask.jpg)

      Figure 9: Task Deletion


## Job Management

Topics covered under job management:
-  [Display All Jobs](#display-all-jobs)
-  [Display Job Details](#display-job-details)
-  [Display Job Output File Information](#display-job-output-file-information)
-  [Navigate Job Information](#navigate-job-information)


### Display All Jobs

To display all jobs:

Navigate to the Job Management page by clicking **VIEW JOBS**.

![Display All Jobs](./images/DisplayAllJobs.jpg)

Figure 10: Display All Jobs

> **Note**  
> Jobs are updated automatically every 10 seconds.

### Display Job Details

To see job details:

- Click on the down carrot in the first column to display the details about that particular input file.
- Click on **Expand All Input File Details** button to view the information about all input files at once.

![Display Filename Jobs](./images/DisplayFilenameJobs.jpg)

Figure 11: Display Filename Jobs

### Display Job Output File Information 

To view the output file(s):

Click on the button in the output file column. A box will appear with all the associated output files.

![Display Output Files Jobs](./images/DisplayOutputfilesJobs.jpg)

Figure 12: Display Output Files Jobs

To view the output file(s) status:

Hover the mouse over the **File Status** for the desired file as shown below.

![Output File Status Tool Tip](./images/OutputFileStatusToolTip.png)

Figure 13 Output File Status Tool Tip

If the status is an error, hovering the mouse over the **Output File Path** will show the error details, as shown below.

![Output File Error Details Tool Tip](./images/OutputFileErrDetailsToolTip.png)

Figure 14: Output File Error Details Tool Tip

### Navigate Job Information

To control how jobs are displayed:

* Enter text in the text box to filter jobs. In this example, the jobs with errors associated with them are displayed.
* Control the number of jobs on the page using pagination provided at the bottom of the page.
* Click on individual columns to sort jobs either in ascending or descending order.

![Display Job Error Details](./images/DisplayJobErrorDetails.jpg)

Figure 15: Display Job Error Details

## Model Upload

The **Upload Models** page provides functionality to upload the AI/ML models to create new pipelines. This feature supports Intel® Geti™ platform models and those served by the OpenVINO™ Model Server (OVMS).

To upload a new model:

1. Complete the following fields with the model details:

    -  **Model Name**: enter Model Name
  
    -  **Model Type**: click to select the type of the model - 
     
        -- **Geti**

        -- **OpenVino Model Server (OVMS)** - *default option*
  
2. Click **Browse** to select the zipped models.

      > **Warning**  
      > - Zip file is **required**. 
      > - Correct directory structure must be followed.
        
3. Click **Save**.

![Upload Model](./images/UploadModel.jpg)

Figure 16: Upload Model

## Dashboard
The dashboard provides additional metrics and monitoring for the system.
Currently, the software uses Grafana, the open-source monitoring and analytics platform,
to display metrics and monitoring in a dashboard.

> **Note**  
> To view the Grafana UI, start the monitoring tech stack as described in [Monitoring Overview](../monitoring/overview.md#to-run).

To use the dashboard: 

Navigate to the **Dashboards** page by clicking on the DASHBOARDS tab in the tab strip at the top.

![Display Grafana UI](./images/DisplayGrafanaUI.jpg)

Figure 17: Display Grafana UI

This page will open a new tab to the Grafana UI default dashboard as shown below.

![Display Grafana Dashboard](./images/DisplayGrafanaDashboard.jpg)

Figure 18: Display Grafana Dashboard



