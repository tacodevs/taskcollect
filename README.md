# TaskCollect
Lightweight, web-based, multiplexed frontend for online educational platforms. 

## Description

Tired of using bloated, buggy, or unintuitive educational platforms? Forced to use them for school or university? Want to throw all of them away and find an alternative approach that truly respects both your privacy and your time?

TaskCollect is a lightweight, web-based frontend that allows you to use existing platforms in a way that best suits your needs. Our goal is for you to stop wasting time managing platforms and to manage your learning instead. To this end, we have designed TaskCollect to be:

  - lightweight, accessible, and fast
  - multiplexed (all your platforms appear as one)
  - easy-to-use, efficient, and productive (based on continuous user feedback)
  - free, libre, and open-source
  - easy to fix, maintain, and improve

**Our team is currently working on TaskCollect v2 which will completely bring the above goals to fruition!**

TaskCollect currently supports the following educational institutions:

  - Glenunga International High School

Is your institution not in the list above? [Contribute][5] today to make it happen!

## Setup

Build dependencies^:
  * Git
  * Make
  * Go 1.20+
  * Dart Sass 1.57+

^ Windows users should install a [Unix shell][9] for Make to function properly.

Clone this Git repository and build TaskCollect from source:

```
git clone https://git.sr.ht/~kvo/taskcollect
cd taskcollect/
make
```

To cross-compile for a specific platform (for a list of supported platforms, run `go tool dist list`):

```
GOOS=linux GOARCH=amd64 make
```

To build release versions for all supported platforms, use the following:

```
make all
```

## Usage

For personal or development purposes, use the following command:

```
make run
```

When deploying in production, use this instead:

```
make deploy
```

The TaskCollect server should now start on TCP port 443 (if deploying) or on port 8080 (for personal use).

## Contributing

TaskCollect development occurs on [SourceHut][1]. Mirrors may exist elsewhere (e.g. the official mirrors on [Codeberg][2] and [GitHub][3]), but contributions will only be accepted through SourceHut.

Confirmed issues and feature requests may be reported on the [issue tracker][4]. If your issue has not been confirmed, please send an email to [taskcollect-discuss][5] to discuss your query first.

Contributions are welcome! You can send us a pull request, or if you are unsure where to start, send an email to [taskcollect-devel][6] and we can help you get started. If your email reveals information about any educational institution, we advise you to send an email to [taskcollect-discuss][5].

All contributors are required to "sign-off" their commits (using `git commit -s`) to indicate that they have agreed to the [Developer Certificate of Origin][8].

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
[3]: https://github.com/tacodevs/taskcollect
[4]: https://todo.sr.ht/~kvo/taskcollect
[5]: mailto:~kvo/taskcollect-discuss@lists.sr.ht
[6]: mailto:~kvo/taskcollect-devel@lists.sr.ht
[7]: https://lists.sr.ht/~kvo/taskcollect-announce
[8]: https://developercertificate.org/
[9]: https://kvo.envs.net/tutorials/win11unix.html
