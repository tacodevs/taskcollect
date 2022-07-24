taskcollect
===========

A web server which multiplexes functionality from educational web platforms for students.

Description
-----------

There is a great number of schools around the world which provide educational resources, homework, methods for communication, etc. through networked technologies, in particular web-based educational platforms. However, due to a number of different factors, schools may employ multiple platforms in their education programs, typically with noticeable overlaps in functionality. In such cases, students have no choice but to wade their way through all the platforms in use by their school in order to retrieve their homework and available educational resources.

The solution to this is TaskCollect: a web server which multiplexes functionality from educational web platforms, providing a clean, simple, intuitive web interface for students.

At the moment, TaskCollect acts as a multiplexed interface for the following platforms:
  * DayMap
  * Google Classroom

Unfortunately, some of the platforms in use by schools (e.g. DayMap) are school-dependent. To counter this, TaskCollect ensures each user is associated with a particular school.

TaskCollect supports users from the following schools:
  * Glenunga International High School

Setup
-----

Build dependencies:
  * Git
  * Go
  * Make

TaskCollect has very simple build and deployment mechanisms. Simply clone this Git repository, enter its `src/` subdirectory, and run `make configure` and `make`:

```
git clone https://codeberg.org/kvo/taskcollect.git
cd taskcollect/src/
make configure
make
```

If all the build dependencies are installed and no errors occur, the folder `prg/` should appear in the root folder of the repository, containing executable programs for all major operating systems and CPU architectures. From here, deployment is easy: simply run the program for your OS and CPU, and the web server will start. For more information about deployment and running of TaskCollect, see `doc/en/cmd/taskcollect`.

TaskCollect is ***not*** production-ready and should not be deployed to the public yet. When it is, TaskCollect and its host system should be protected by a strong firewall to prevent damage from bad actors.

Contribute
----------

TaskCollect, in its current state, is much like a newborn child, and still needs to grow and develop into a secure, robust program. For that we need people testing TaskCollect, finding bugs/issues, providing fixes and suggestions. If you have the time, please consider making a contribution, however small, to TaskCollect development.

Issues can be reported on the Codeberg issue tracker. If you need to discuss something or ask questions about TaskCollect, this can be done through Matrix: #taskcollect:matrix.org

It should be noted that the issue tracker is for issues and feature requests *only*, and the Matrix room is also not a place for general discourse.
