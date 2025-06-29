# **Project Requirements Document: MyGuy Website**

MyGuy is a task marketplace platform allowing users to create tasks for others to complete, apply for tasks, communicate about requirements, and manage the entire process from creation to completion

| Requirement ID | Description               | User Story                                                                                       | Expected Behavior/Outcome                                                                                                     |
|-----------------|---------------------------|--------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| FR001          | User Registration & Authentication | As a user, I want to register myself on the system and be able to login.   | The system should provide a streamlined registration process and secure login functionality.  |
| FR002          | Task Creation | As a user, I want to create a new task that someone else can complete for me. | The system should provide a clear interface for users to create new tasks with all necessary fields (title, description, deadline, etc.). |
| FR003          | Task Requirements Management | As a user, I want to add and update requirements and descriptions for my tasks. | The system should provide editable fields for task details with save functionality that immediately updates the task information. |
| FR004          | Personal Task Dashboard | As a user, I want a dashboard where I can see all tasks I've created or applied for. | The system should display a personalized dashboard showing tasks categorized by status (created, applied for, in progress, completed).  |
| FR005          | Browse Available Tasks |As a user, I want to browse open tasks created by other users. | The system should provide a searchable, filterable marketplace view of all available tasks created by other users. |
| FR006          |Task Communication  | As a user, I want to ask questions about tasks I'm interested in completing. | The system should provide a dedicated messaging system for each task where interested users can communicate with task owners. |
| FR007          | Task Assignment | As a task creator, I want to assign my tasks to users who have applied to complete them. | The system should provide task owners with a list of applicants and the ability to select and assign a task to their chosen applicant.  |
| FR008          | Fee Negotiation | As a task applicant, I want to request a fee from the task owner for completing their task. | The system should provide a fee proposal feature within the application process, allowing applicants to specify their requested payment. |
| FR009          | Application Management | As a task creator, I want to approve or decline applications for my tasks |The system should notify task owners of new applications and provide accept/decline options with appropriate notifications to applicants. |
| FR010          |Task Status Updates | As a user, I want to update the status of tasks I've created or been assigned. | The system should provide status options (pending, in-progress, completed) that update task records and notify relevant parties. |
| FR011          | Task Rating System  | As a task owner, I want to rate and review completed tasks. | The system should prompt task owners to provide ratings and feedback when a task is marked as completed. |
| FR012          | User Profiles       | As a user, I want to view profiles of other users to evaluate their reliability  | The system should display user profiles with relevant information such as completed tasks, ratings, and reviews.                                                               |
| FR013          | Task Deletion          |As a task creator, I want to remove tasks I no longer need completed. | The system should provide task creators with the option to delete their tasks, with appropriate confirmation steps and notifications to any affected.


Non-Functional Requirements

Security: The system shall protect user data and ensure secure transactions.
Usability: The interface shall be intuitive and accessible across devices.
Performance: Page load times should not exceed 2 seconds under normal conditions.
Reliability: The system should maintain 99.9% uptime.
Scalability: The architecture should support growing user numbers without performance degradation.