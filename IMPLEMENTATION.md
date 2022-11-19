# taskcollect: Implementation Documentation

This document outlines how features are implemented in TaskCollect. 


## Database

TaskCollect uses Redis 7 as its primary user credentials database. 

User credentials are stored using Redis hashes, using the following key reference:
```
school:<school>:studentID:<ID>
```
where ``<school>`` is the school name and ``<ID>`` is the unique student ID that the school provides. This allows TaskCollect to support multiple schools if needed. 

Currently, the following information on students are stored:
- ``token``: TaskCollect session token
- ``school``: school name
- ``username``: username (student ID)
- ``password``: password for the school's SAML services
- ``daymap``: DayMap session token
- ``gclass``: Google Classroom auth token

Furthermore, TaskCollect also creates an index of current session tokens, using the following key reference, where ``<token>`` is the current session token.
```
studentToken:<token>
```
``studentToken:<token>`` also stores a Redis hash and contains the following information:
- ``studentID``: username (student ID)
- ``school``: school name

Currently, TaskCollect's session tokens last for 7 days and by extension, this data is only stored for 7 days as well. 

By using this index rather than ``school:<school>:studentID:<ID>``, it allows for faster look-ups using the TaskCollect token as a reference, rather than the alternative of looping through each student ID in an attempt to fetch the session token.

A list of students per school is also stored via ``school:<school>:studentList``, which is a Redis set containing the unique student IDs for each school. Currently, it is not used for anything, although it may be useful in the future.
