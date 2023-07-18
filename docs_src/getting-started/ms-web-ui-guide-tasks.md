# Web User Guide - Task Management

## Display All Tasks

The default landing page is **Task Management**. Clicking the **Create/Modify Tasks** at the top-left of the navigation bar also displays the **Task Management** page. This page will display all the tasks currently available in the system.


<figure class="figure-image">
<img src="..\images\DisplayAllTasks.jpg" alt="Figure 1: Display all tasks">
<figcaption>Figure 1: Display All Tasks</figcaption>
</figure>

The UI, shown in Figure 1, offers different ways to look at the current tasks:

1. **Filter:** enter text in the textbox to filter tasks.

2. **Sort:** click on individual columns to sort tasks either in ascending or descending order.

3. **View pages:** use pagination, provided at the bottom of the page, to display a set number of tasks on the page.

## Create Task

To create a task: 

1. From Task Management page, select **Add Task**. 

    <figure class="figure-image">
    <img src="..\images\CreateTask.jpg" alt="Figure 2: Create tasks">
    <figcaption>Figure 2: Create Task</figcaption>
    </figure>

2. Complete the following fields with the task details:
        
    !!! Note
        Regular expressions are not supported.

        For example, in **Job Selector** below, use of **contains** with "\*.tiff" will filter for tasks with the literal substring "\*.tiff".

    -  **Description**: enter Task Description
  
    -  **Pipeline**: click to select the Model Execution Pipeline from the dropdown
  
    -  **Job Selector**: click to select between the two checkboxes -
  
        --  **matches**: filter jobs based on filename that exactly match “filename” field value from below
    
        --  **contains**: filter jobs based on filename that contain “filename” field value from below
    
    -  **Filename**: enter input image filename   

    -  **Model Parameters**: enter parameter details for the model, it should adhere to the following json format, including quotes – {“parameter” : ”value”}

    <figure class="figure-image">
    <img src="..\images\FillFormTask.jpg" alt="Figure 3: Task Form">
    <figcaption>Figure 3: Task Form</figcaption>
    </figure>

3. Click **Save**.

## Update Task

1. Click on **Update** for the task that needs to be modified:

    <figure class="figure-image">
    <img src="..\images\UpdateTask.jpg" alt="Figure 4: Update Task">
    <figcaption>Figure 4: Update Task</figcaption>
    </figure>

2. Modify the appropriate field (i.e., **Description** modified)

    <figure class="figure-image">
    <img src="..\images\ModifyFieldTask.jpg" alt="Figure 5: Modify Field Task">
    <figcaption>Figure 5: Modify Field Task</figcaption>
    </figure>


3. Click **Save**.
4. Confirm that the changes are saved as shown below:

    <figure class="figure-image">
    <img src="..\images\UpdatedTask.jpg" alt="Figure 6: Updated Task">
    <figcaption>Figure 6: Updated Task</figcaption>
    </figure>


## Delete Task

1. Select one or more checkboxes next to the entry/entries to delete.
2. Click **Delete Selected**. 
3. Confirm the deletion:
    
    <figure class="figure-image">
    <img src="..\images\DeleteTask.jpg" alt="Figure 7: Task Deletion">
    <figcaption>Figure 7: Task Deletion</figcaption>
    </figure>

## Next up

[Web UI Guide - Jobs](./ms-web-ui-guide-jobs.md)

BSD-3 License: See [License](../LICENSE.md).
