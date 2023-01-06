# taskcollect
A web server that multiplexes functionality from educational web platforms for students.

## Status

**Version 1.0.0 is set to be released on 14 January 2023.**

Internal testing (testing by the developers) will begin on 9 January at the latest. External testing will probably begin on 20 January but can begin a bit earlier/later if necessary. The version 1.0.0 release date will not be changing and can be safely relied upon.

## Description

There is a great number of schools around the world that provide educational resources, homework, methods for communication, etc. through networked technologies, in particular web-based educational platforms. However, due to several different factors, schools may employ multiple platforms in their education programs, typically with noticeable overlaps in functionality. In such cases, students have no choice but to wade their way through all the platforms in use by their school to retrieve their homework and available educational resources.

The solution to this is TaskCollect: a web server that multiplexes functionality from educational web platforms, providing a clean, simple, intuitive web interface for students.

At the moment, TaskCollect acts as a multiplexed interface for the following platforms:
  * DayMap
  * Google Classroom

Unfortunately, some of the platforms in use by schools (e.g. DayMap) are school-dependent. To counter this, TaskCollect ensures each user is associated with a particular school.

TaskCollect currently supports users from the following schools:
  * Glenunga International High School

## Setup

### Building
Build dependencies:
  * Git
  * Go 1.18+
  * Python 3.9+ (for the build script)
  * Sass 1.57+

TaskCollect has very simple build and deployment mechanisms. Simply clone this Git repository, enter into its `src/` subdirectory, and run `python3 build.py -u`:

```
git clone https://codeberg.org/kvo/taskcollect.git
cd taskcollect/src/
python3 build.py -u
```

If all the build dependencies are installed and no errors occur, the folder `prg/` should appear in the root folder of the repository, containing the executable program for the current system. The `-u` flag in the build script additionally copies across the assets that are required to run TaskCollect from the `res/` folder to `$home/res/taskcollect`.

To build executable programs for all major operating systems and CPU architectures, run `python3 build.py all`

### Deployment

Deployment dependencies:
  * Redis 7

From here, deployment is simple:

  1. Copy the contents of the `res/` folder into `$home/res/taskcollect/` where `$home` is the current user's home directory. You can either do this manually or use the build script. Invoke the build script using the `-u` flag to both build TaskCollect and copy the contents of the `res/` folder, or use the `-U` flag to copy the `res/` folder without building.

  2. Obtain a Google OAuth 2.0 client ID and save it to `$home/res/taskcollect/` (see `doc/en/cmd/taskcollect` for more info)

  3. Run the program for your OS and CPU, and the web server will start, asking you for a passphrase at first.

  4. If running for the first time, you will also need to set up a Redis server. If it is not the first time running TaskCollect or the credentials database has been imported from another TaskCollect session, then set up the Redis server as you normally would, then run TaskCollect.


TaskCollect is ***not*** production-ready and should not be deployed to the public yet. When it is, TaskCollect and its host system should be protected by a strong firewall to prevent damage from bad actors.

## Contribute

TaskCollect, in its current state, is much like a newborn child and still needs to grow and develop into a secure, robust program. For that we need people testing TaskCollect, finding bugs/issues, and providing fixes and suggestions. If you have the time, please consider contributing, however small, to TaskCollect development.

Issues can be reported on the Codeberg issue tracker. For discussions and questions regarding TaskCollect, there are two mediums:

  * Public mailing list (~kvo/taskcollect@lists.sr.ht)
  * Matrix (#taskcollect:matrix.org)

It should be noted that the issue tracker is for issues and feature requests *only*, and neither the public mailing list nor the Matrix room are places for general discourse. The purpose of the Matrix room is **only for technical discussion** about the development of TaskCollect. **All general queries should be sent to the mailing list.** Questions about technical matters are also welcome on the mailing list, but general enquiries are not welcome in the Matrix room.

Although the mailing list is public, the mailing list archives are not accessible to the general public as schools and those who work, teach, or learn therein may potentially ask school-specific questions about TaskCollect.

## Future directions

Currently, as TaskCollect is slowly heading to version 1.0.0, it is evident that many other useful features could be added in the future. However, the scope of TaskCollect's aims for version 1.0.0 is rather conservative to emphasise robustness, security, and stability. Though in the future, the following features could be added (potentially through a bounty program).

Support for:

  * Edpuzzle
  * Stile
  * InThinking
  * Kognity

Additional feature tabs to TaskCollect:

  * Organisation (calendar-oriented organisation space)
  * Communication (emails, messages, etc.)
  * Grades (grade summaries per term, with a graph)
