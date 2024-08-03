# Contributing guidelines

## Committing changes

### Commit messages

Before committing changes, you must make sure that (1) the code in each of your
commits compiles, and (2) your commit messages follow the following style:

```
changed_file: commit message
```

Example:

```
src/main.go: create main file
```

Your commit message should finish the following sentence: *This commit will ...*

If you have two or three files changed, you can list them using commas as long as
the resulting prefix is reasonably short:

```
license, readme: fix typos
```

Otherwise, if multiple files have changed, specify the folder that has changed
instead:

```
src: import existing codebase
```

If multiple folders have changed, specify multiple folders (i.e. `doc, src:`),
or, if too many files/folders have changed, specify `all`:

```
all: rewrite web app as gui app
```

If the change is significant, you should specify more details in the extended
commit message:

```
all: rewrite web app as gui app

Rewrote the entire web server as a GUI app. Removed the dependency on
PostgreSQL; user settings are now stored in the user's app data
folder. Authentication details are no longer handled internally - they
are managed externally by the user's preferred password manager or
keyring program.
```

Make sure your extended commit message is hard-wrapped to 70 columns!

### Signing off commits

You must sign off your commits with `git commit -s` to signify that your commits
were created in whole by you and you have the right to submit it under the license
of this project.

Essentially, if you were committing with this before:

```
git commit -m "this is my commit message"
```

All that is required is for you to do this instead:

```
git commit -sm "this is my commit message"
```

By signing off your commits, you indicate that you have agreed to the
[Developer Certificate of Origin][2].

### Sending changes

You must use [git send-email][1] to submit your changes. In particular this
means your contributions will be sent as a patchset, and will be sent via email.
You must make sure that you send your patch from an email address that you are
comfortable with being publicly visible.

Patches must be sent to <~kvo/taskcollect-devel@lists.sr.ht>. Please make sure
that your patch email subject follows the same naming conventions as those for
commit messages.

If for some reason you need to send your contribution from an email that you do
not want to be public, you should send your patch to [taskcollect-discuss][3]
and use the [`--from`][4] flag to specify a public email address that you
control (e.g. your anonymous GitHub noreply address). If you choose this option,
you will need to provide evidence that you control the alternative email address
you provide.

## Platform support

### Adding support for a new institution

To add support for a new institution, you will need to:

  1. Add the institution to the alphabetically ordered login page selection:

     ```
     <select id="school" name="school">
         <option value="gihs">Glenunga International High School</option>
         <option value="newschool">Newly Added Institution</option>
         <option value="uofa">University of Adelaide</option>
     </select><br>
     ```

  1. Update the list of enrolled institutions at the start of `server.Run`:

     ```go
     enrol("gihs", "newschool", "uofa")
     ```

  2. Add institution timezone and login configurations to `server.auth`:

     ```go
     case "newschool":
     	user.Timezone = time.UTC
     	err = schools["newschool"].Auth(&user)
     	if err != nil {
     		return site.User{}, errors.New("", err)
     	}
     ```

  3. Configure the new institution's supported platfoms in `server.enrol`:

     ```go
     case "newschool":
     	schools["newschool"] = site.NewMux()
     	schools["newschool"].AddAuth(newsite.Auth)
     	schools["newschool"].AddLessons(newsite.Lessons)
     	schools["newschool"].SetReports(newsite.Reports)
     ```


[1]: https://git-send-email.io
[2]: https://developercertificate.org/
[3]: https://lists.sr.ht/~kvo/taskcollect-discuss
[4]: https://git-scm.com/docs/git-send-email#Documentation/git-send-email.txt---fromltaddressgt
