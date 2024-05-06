# TaskCollect
Lightweight, web-based, multiplexed frontend for online educational platforms. 

## Description

Many educational institutions around the world use web-based educational platforms. However, these platforms may have numerous problems — they may be bloated, unreliably engineered, unintuitive, and so on. Furthermore, they may have duplicate functionality, which would make managing learning across multiple platforms extremely difficult. With each new confusing buggy platform, the amount of time wasted trying to manage learning outgrows the amount of learning actually done.

The solution to this problem is TaskCollect: a lightweight, web-based frontend that multiplexes functionality from educational web platforms, in a single, reliable, and intuitive web interface.

TaskCollect currently supports the following institutions and platforms:

  - Glenunga International High School
     - Daymap
     - Google Classroom (deprecated)

## Setup

Build dependencies:
  * Git
  * Go 1.20+
  * Python 3.9+ (build, test, etc. scripts)
  * Dart Sass 1.57+

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

Use `./run.py` to start the TaskCollect server (`./run.py -w` for development purposes).

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
