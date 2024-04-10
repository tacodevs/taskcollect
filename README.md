# TaskCollect
A web server that multiplexes functionality from educational web platforms for students.

## Description

There are a great number of educational institutions around the world that provide educational resources, homework, methods for communication, etc. through networked technologies, in particular web-based educational platforms. However, due to many factors, institutions may employ multiple platforms in their education programs, typically with noticeable overlaps in functionality. In such cases, students have no choice but to manage the large number of platforms in use by their institution to retrieve their homework and available educational resources. This inconvenience could quickly become a serious problem if the number of platforms in use at the institution drastically increases.

The solution to this problem is TaskCollect: a web server that multiplexes functionality from educational web platforms, which provides a single, reliable, and intuitive web interface for students.

At the moment, TaskCollect acts as a multiplexed interface for the following platforms:
  * Daymap
  * Google Classroom (via the `google-api` branch)

Unfortunately, some of the platforms in use by institutions (e.g. Daymap) are institution-dependent. To counter this, TaskCollect ensures each user is associated with a particular institution.

TaskCollect currently supports users from the following institutions:
  * Glenunga International High School

## Setup

Build dependencies:
  * Git
  * Go 1.20+
  * Python 3.9+ (build, test, etc. scripts)
  * Sass 1.57+ (must be installed from npm)

Clone this Git repository and run the build script with `./build.py -u`:

```
git clone https://git.sr.ht/~kvo/taskcollect
cd taskcollect/
./build.py -u
```

For more information on how to use the build script, run `./build.py help`.

## Usage

Run-time dependencies:
  * Redis 7

## Contributing

TaskCollect development occurs on [SourceHut][1]. Mirrors may exist elsewhere (e.g. the official mirrors on [Codeberg][2] and [GitHub][3]), but contributions will only be accepted through SourceHut.

Confirmed issues and feature requests may be reported on the [issue tracker][4]. If your issue has not been confirmed, please send an email to [taskcollect-discuss][5] to discuss your query first.

Contributions are welcome! You can send us a pull request, or if you are unsure where to start, or send an email to [taskcollect-devel][6] and we can help you get started. If your email reveals information about any educational institution, we advise you to send an email to [taskcollect-discuss][5].

All contributors are required to "sign-off" their commits (using `git commit -s`) to indicate that they have agreed to the [Developer Certificate of Origin][8].

Tests can be run using `test.py`. For more information, run `./test.py help`.

## Moldy spots

TaskCollect currently has a lot of moldy spots that need to be fixed:
  * Bad UX due to bad UI (see the issue tracker)
  * It does too many tasks at once (scraping, muxing, UI, auth, etc.)
  * Some feature requests need a persistent-memory database
  * It's a web server (GUI app would be nicer)
  * The codebase is ugly (refactor required)
  * The official website is on Codeberg (slow) and badly designed

We will be releasing a proposal for TaskCollect v2 which will address these moldy spots. Keep an eye out for the official announcement.

## Announcements

To keep up with official TaskCollect announcements, subscribe to the announcements mailing list by sending an email to this email address: <~kvo/taskcollect-announce+subscribe@lists.sr.ht>

For all previous announcements, [see the archives][7].

## Contact us

If you have a query, please send an email to the official mailing list: <~kvo/taskcollect-discuss@lists.sr.ht>

The list archives are hidden to protect the privacy of educational institutions.


[1]: https://sr.ht/~kvo/taskcollect
[2]: https://codeberg.org/kvo/taskcollect
[3]: https://github.com/kv-o/taskcollect
[4]: https://todo.sr.ht/~kvo/taskcollect
[5]: mailto:~kvo/taskcollect-discuss@lists.sr.ht
[6]: mailto:~kvo/taskcollect-devel@lists.sr.ht
[7]: https://lists.sr.ht/~kvo/taskcollect-announce
[8]: https://developercertificate.org/
