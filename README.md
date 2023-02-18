# TaskCollect
A web server that multiplexes functionality from educational web platforms for students.

## Description

There are a great number of schools around the world that provide educational resources, homework, methods for communication, etc. through networked technologies, in particular web-based educational platforms. However, due to several different factors, schools may employ multiple platforms in their education programs, typically with noticeable overlaps in functionality. In such cases, students have no choice but to wade their way through all the platforms in use by their school to retrieve their homework and available educational resources.

The solution to this is TaskCollect: a web server that multiplexes functionality from educational web platforms, providing a clean, simple, intuitive web interface for students.

At the moment, TaskCollect acts as a multiplexed interface for the following platforms:
  * Daymap
  * Google Classroom

Unfortunately, some of the platforms in use by schools (e.g. Daymap) are school-dependent. To counter this, TaskCollect ensures each user is associated with a particular school.

TaskCollect currently supports users from the following schools:
  * Glenunga International High School

## Setup

### Building
Build dependencies:
  * Git
  * Go 1.18+
  * Python 3.9+ (build script)
  * Sass 1.57+

TaskCollect has very simple build and deployment mechanisms. To build everything automatically across multiple platforms (including managing resource dependencies), a Python build script is used.

Simply clone this Git repository and run `python3 build.py -u` (or `py build.py -u` on Windows):

```
git clone https://codeberg.org/kvo/taskcollect.git
cd taskcollect
python3 build.py -u
```

If all the build dependencies are installed and no errors occur, the folder `prg/` should appear in the root folder of the repository, containing the executable program for the current system. The `-u` flag in the build script additionally copies across the assets that are required to run TaskCollect from the `res/` folder to `$home/res/taskcollect`. For more information on how to use the build script, run `python3 build.py help`

To build executable programs for all major operating systems and CPU architectures, run `python3 build.py all`. Be aware that this may take a while.

### Deployment

Deployment dependencies:
  * Redis 7

From here, deployment is simple:

  1. Copy the contents of the `res/` folder into `$home/res/taskcollect/` where `$home` is the current user's home directory. You can either do this manually or use the build script. Invoke the build script using the `-u` flag to both build TaskCollect and copy the contents of the `res/` folder, or use the `-U` flag to copy the `res/` folder without building.

  2. Obtain a Google OAuth 2.0 client ID and save it to `$home/res/taskcollect/` (see `doc/en/cmd/taskcollect` for more info)

  3. Run the program for your OS and CPU.

  4. If running for the first time, you will also need to set up a Redis server. You must ensure you are Redis 7 as no other version will work. Ensure that the Redis server is running and unexposed to the network before you run TaskCollect.

  5. TaskCollect will ask you for a passphrase to the running Redis server. If you have not set up a password for Redis, it is sufficient to press enter/return at this prompt, and the TaskCollect web server will start. Otherwise, enter the password you configured for Redis to start the web server.

TaskCollect and its host system should be protected by a strong firewall to prevent damage from bad actors. In particular, the firewall should prevent overly frequent requests to TaskCollect, as some APIs that TaskCollect uses enforce a stringent request rate limit. Those deploying TaskCollect should additionally request higher request rate limits for the Google Classroom API to avoid having "ratelimit exceeded" errors.

As of v1.0.0, TaskCollect is considered production-ready and its public interface is stable (see `IMPLEMENTATION.md` for more information). If you are deploying TaskCollect, you should consult [the Releases tab](https://codeberg.org/kvo/taskcollect/releases) for official releases. Where possible, select the latest available release in order to ensure you are running the most secure and issue-free version available.

Easier installation and update mechanisms for TaskCollect will be developed in the near future to make these processes easier and smoother for deployers.

## Testing

Tests can be run using `test.py`. Invoke this script without any arguments to test all packages automatically. For more information, use `python3 test.py help` (`py test.py help` on Windows).

## Contribute

Issues can be reported on the Codeberg issue tracker. You can ask detailed questions (including those that might reveal school-specific information) on the public mailing list (the archives are private to ensure school privacy). For discussion regarding TaskCollect development, an IRC channel is also available.

The links to these mediums are provided below:

  * Codeberg issue tracker (https://codeberg.org/kvo/taskcollect/issues)
  * Public mailing list (~kvo/taskcollect@lists.sr.ht)
  * IRC (#taskcollect on irc.libera.chat)

It should be noted that the issue tracker is for issues and feature requests *only*, and neither the public mailing list nor the IRC channel are places for general discourse. The purpose of the IRC channel is **only for technical discussion** about the development of TaskCollect. **All general queries should be sent to the mailing list.** Questions about technical matters are also welcome on the mailing list, but general enquiries are not welcome in the IRC channel.

Although the mailing list is public, the mailing list archives are not accessible to the general public as schools and those who work, teach, or learn therein may potentially ask school-specific questions about TaskCollect.

## Future directions

Despite the v1.0.0 release, it is evident that many other useful features (and fixes) could be added in the future. However, the scope of TaskCollect's aims for version 1.0.0 has been rather conservative to emphasise robustness, security, and stability. In the future, the following features could be added (potentially through a bounty program).

Support for:

  * Edpuzzle
  * Stile
  * InThinking
  * Kognity

Additional feature tabs to TaskCollect:

  * Organisation (calendar-oriented organisation space)
  * Communication (emails, messages, etc.)
  * Grades (grade summaries per term, with a graph)
