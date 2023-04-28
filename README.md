# TaskCollect
A web server that multiplexes functionality from educational web platforms for students.

## Description

There are a great number of schools around the world that provide educational resources, homework, methods for communication, etc. through networked technologies, in particular web-based educational platforms. However, due to many factors, schools may employ multiple platforms in their education programs, typically with noticeable overlaps in functionality. In such cases, students have no choice but to manage the large number of platforms in use by their school to retrieve their homework and available educational resources. This inconvenience could quickly become a serious problem if the number of platforms in use at the school drastically increases.

The solution to this problem is TaskCollect: a web server that multiplexes functionality from educational web platforms, which provides a single, reliable, and intuitive web interface for students.

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
  * Go 1.20+
  * Python 3.9+ (build script)
  * Sass 1.57+

TaskCollect has relatively simple build and deployment mechanisms. The TaskCollect codebase ships with a Python build script to build TaskCollect from source.

Clone this Git repository and run `./build.py -u` (or `py build.py -u` on Windows):

```
git clone https://codeberg.org/kvo/taskcollect.git
cd taskcollect/
./build.py -u
```

If all the build dependencies are installed and no errors occur, the folder `prg/` should appear in the root folder of the repository, containing the executable program for the current system. The `-u` option in the build script (as used in this example) additionally copies the assets TaskCollect requires from the `res/` folder to `$home/res/taskcollect`. For more information on how to use the build script, run `./build.py help` (or the equivalent command on Windows).

To build executable programs for all major operating systems and CPU architectures, run `./build.py all`. Be aware that this may take a while.

### Deployment

Deployment dependencies:
  * Redis 7

Although building TaskCollect from source is simple, deployment is not (unfortunately). TaskCollect is currently dependent on external APIs and has deployment dependencies which must be installed separately alongside TaskCollect. The TaskCollect development team will address this inconvenience in the future, but for the moment, deployers must take the following steps:

  1. Ensure that the contents of the repository's `res/` folder are copied into `$home/res/taskcollect/` where `$home` is the current user's home directory. This can be done automatically by invoking the build script with the `-U` option, which copies `res/` without building TaskCollect. (Use `-u` if both building and copying `res/`)

  2. Obtain a Google OAuth 2.0 client ID and save it to `$home/res/taskcollect/` (see `doc/en/cmd/taskcollect` for more info)

  3. If running for the first time, you will also need to set up a Redis server. You must ensure you are Redis 7 as no other version will work. Ensure that the Redis server is running and unexposed to the network before you run TaskCollect.

  4. Run the program for your OS and CPU (e.g. `prg/linux/amd64/taskcollect`).

  5. TaskCollect will ask you for a passphrase to the running Redis server. If you have not set up a password for Redis, it is sufficient to press enter/return at this prompt, and the TaskCollect web server will start. Otherwise, enter the password you configured for Redis to start the web server.

TaskCollect and its host system should be protected by a strong firewall to prevent damage from bad actors. In particular, the firewall should prevent overly frequent requests to TaskCollect, as some APIs that TaskCollect uses enforce a stringent request rate limit. Those deploying TaskCollect should additionally request higher request rate limits for the Google Classroom API to avoid having "ratelimit exceeded" errors.

As of v1.0.0, TaskCollect is considered production-ready and its public interface is stable (see `misc/info/impl.md` for more information). If you are deploying TaskCollect, you should consult [the Releases tab](https://codeberg.org/kvo/taskcollect/releases) for official releases. Where possible, select the latest available release in order to ensure you are running the most secure and issue-free version available.

Easier installation and update mechanisms for TaskCollect will be developed in the future to make these processes easier and smoother for deployers. Furthermore, it is a regrettable decision for the TaskCollect resources folder to be located in the user's home folder — one which will be corrected in a future version of TaskCollect.

## Testing

Tests can be run using `test.py`. Invoke this script without any arguments to test all packages automatically. For more information, use `./test.py help` (or `py test.py help` on Windows).

## Contribute

Issues can be reported on the Codeberg issue tracker. You can ask detailed questions (including those that might reveal school-specific information) on the public mailing list (the archives are private to maintain school privacy).

The links to these mediums are provided below:

  * Codeberg issue tracker (https://codeberg.org/kvo/taskcollect/issues)
  * Public mailing list (~kvo/taskcollect@lists.sr.ht)

It should be noted that the issue tracker is for issues and feature requests *only*, and the public mailing list is *not* a place for off-topic discourse.

Although the mailing list is public, the mailing list archives are not accessible to the general public as schools and those who work, teach, or learn therein may potentially ask school-specific questions about TaskCollect.

## Announcements

To keep up with official TaskCollect announcements, subscribe to the announcements mailing list by sending an email to this email address: ~kvo/taskcollect-announce+subscribe@lists.sr.ht

For all previous announcements, [see the archives][1].


[1]: https://lists.sr.ht/~kvo/taskcollect-announce
