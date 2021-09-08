# taskcollect
An online platform to view data from all of your GIHS online platforms in one place.

## Table of contents
* [Releases](#releases)
* [Requirements](#requirements)
* [Usage](#usage)
* [Help and documentation](#help-and-documentation)
* [For developers](#for-developers)

## Releases
Installers for stable and beta releases of TaskCollect *will* be available in the [releases](https://github.com/taskcollect/taskcollect/releases) tab. For installation instructions see [Installation](#installation).

*At the current moment, TaskCollect is a work in progress. Do not expect any functionality yet.*

**Latest stable release:** none.<br>
**Latest beta release:** none.

## Requirements
* A graphical web browser that can render HTML and CSS.
* python3
* python3-pip
* Python modules for the Google Classroom API (install with `pip install --upgrade google-api-python-client google-auth-httplib2 google-auth-oauthlib`)
* Python modules for DayMap (install with `pip install --upgrade requests requests_ntlm lxml`)

## Usage
Run `srv.py` with Python 3. The web server will start up on [http://localhost:1111](http://localhost:1111).

## Help and documentation
All help and documentation is *supposed to be* available in the 'Help' tab in TaskCollect, or, in `help.html` and the `docs` folder. *In reality there's basically nothing except for the license and a list of active bugs.*

## For developers
*This section will be populated at a later date.*