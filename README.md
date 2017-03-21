# Lataa

Lataa is a cli tool, built in [Go](https://golang.org), used to upload local VCL files to a [Fastly CDN](https://www.fastly.com/) account.

> lataa is "upload" in Finnish

If you require a cli tool for diffing your local VCL files against a remote version within a Fastly account, then see [Ero](https://github.com/Integralist/ero)

* [Why?](#why)
* [Installation](#installation)
* [Usage](#usage)
* [Snippets](#snippets)
* [Logic Flow](#logic-flow)
* [Example](#example)

## Why?

I already had a tool for diffing local VCL files against remote versions stored within our Fastly CDN account (called [Ero](https://github.com/integralist/ero)) and I realised there were still two other minor annoyances for me when dealing with the Fastly UI:

1. I needed to always have my phone on me for two-factor authentication
2. the process of cloning, selecting/uploading individual files, diffing them and finally activating was tedious.

As I already had the `ero` cli tool, I decided I would utilise a similar design with the lataa cli tool (e.g. utilise the same environment variables, similar flags and similar concurrency design) so although they were separate tools they would still have a consistency between them (I say that because I didn't want one monolithic binary and preferred to have two separate code bases for easier maintainability).

## Installation

```bash
go get github.com/integralist/lataa
```

## Usage

```bash
lataa -help

  -activate-version string
        specify Fastly service 'version' to activate
  -clone-version string
        specify Fastly service 'version' to clone from before uploading to
  -dir string
        vcl directory to upload files from (default "VCL_DIRECTORY")
  -get-settings string
        get settings (Default TTL & Host) for specified Fastly service version (version number or latest)
  -get-latest-version
        get latest Fastly service version and its active status
  -get-version-status string
        retrieve status for the specified Fastly service 'version'
  -help
        show available flags
  -match string
        regex for matching vcl directories (will also try: VCL_MATCH_DIRECTORY)
  -service string
        your service id (fallback: FASTLY_SERVICE_ID) (default "FASTLY_SERVICE_ID")
  -skip string
        regex for skipping vcl directories (will also try: VCL_SKIP_DIRECTORY) (default "^____")
  -token string
        your fastly api token (fallback: FASTLY_API_TOKEN) (default "FASTLY_API_TOKEN")
  -upload-version string
        specify non-active Fastly service 'version' to upload to
  -use-latest-version
        use latest version to upload to (presumes not activated)
  -version
        show application version
```

## Snippets

None of the following snippets will work without one of these listed flags:

* `-clone-version`
* `-upload-version`
* `-use-latest-version`

Although, any snippet that doesn't specify a `-service` or `-token` flag, does presume the equivalent environment variable has been set in its place.

> See also the "[Logic Flow](#logic-flow)" section below for more details

Specify credentials via cli flags:

```bash
lataa -service 123abc -token 456def
```

> If no flags provided, fallback to environment vars:  
> `FASTLY_SERVICE_ID` and `FASTLY_API_TOKEN`

Specify which nested directories you want to upload VCL files from:

```bash
lataa -match 'foo|bar'
```

> Note: .git directories are automatically ignored  
> If no flag provided, we'll look for the environment var:  
> `VCL_MATCH_DIRECTORY`

Specify which nested directories you want to skip when looking for VCL files to upload:

```bash
lataa -skip 'foo|bar'
```

> If no flag provided, we'll look for the environment var:  
> `VCL_SKIP_DIRECTORY`

---

The following snippets do _not_ require any additional flags.

e.g. they do not also require `-clone-version`, `-upload-version` or `-use-latest-version` (where as the above snippets did).

Although these following snippets do still need the `-service` and `-token` flags (or the appropriate environment variables to be set) so that you can authenticate with Fastly.

---

View latest version and its status:

```bash
lataa -get-latest-version

Latest service version: 123 (already activated)
```

View the status of a specific version:

```bash
lataa -get-version-status 124

Service 'abc' version '124' is 'not activated'
```

Here is that example again but when we specify an incorrect version:

```bash
lataa -get-version-status 999

There was a problem getting version 999

404 - Not Found
Message: Record not found
Detail: Couldn't find Version '[abc, 999, 0000-00-00 00:00:00]'
```

Activate a specific version:

```bash
lataa -activate-version 124 

Service 'abc' now has version '124' activated
```

View the default TTL and Host settings information:

```bash
lataa -get-settings 123

Default Host:
Default TTL: 3600 (seconds)
```

Here is another example of viewing the settings for the latest service version:

```bash
lataa -get-settings latest

Default Host:
Default TTL: 3600 (seconds)
```

## Logic Flow

The application has a specific set of flow controls, which it uses in order to determine where to upload the necessary VCL files to.

If the user doesn't state a version to clone from (e.g. using the `-clone-version` flag), then we'll check to see if they specified an _existing_ version to upload to instead (e.g. using the `-upload-version` flag). 

> Note: the version specified must not be activated  
> Otherwise we'll error appropriately

If the user doesn't use either of those flags, then we'll presume they want to clone the _latest_ service version available and upload the VCL files into this newly cloned version. 

Finally, if the user provides the `-use-latest-version` flag, then we wont clone from the latest service version, but will instead use the latest version for uploading to (as long as it's not already activated; otherwise we'll error appropriately).

## Example

The following example execution has presumed the use of the environment variables: `VCL_DIRECTORY`, `FASTLY_API_TOKEN`, `FASTLY_SERVICE_ID` to keep the length of the command short.

When uploading files into a specific service version, we'll attempt to 'create' the file first. If that fails we'll presume it's because the file already exists and so we'll attempt to update the specified file instead.

In this example, the latest version is not activated and all the VCL files I'm uploading I know to already exist within that version. So I _expect_ the 'creation' of the VCL files to fail (which they do), and subsequently it'll fallback to trying to 'update' the list of VCL files instead (which, as you'll see, succeeds).

```
$ lataa -match www -use-latest-version

There was an error creating the file 'detect_edition':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'detect_edition'
We'll now try updating this file instead of creating it

There was an error creating the file 'ab_tests_config':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'ab_tests_config'
We'll now try updating this file instead of creating it

There was an error creating the file 'ab_tests_callback':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'ab_tests_callback'
We'll now try updating this file instead of creating it

There was an error creating the file 'set_country_cookie':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'set_country_cookie'
We'll now try updating this file instead of creating it

There was an error creating the file 'ab_tests_recv':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'ab_tests_recv'
We'll now try updating this file instead of creating it

There was an error creating the file 'main':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'main'
We'll now try updating this file instead of creating it

There was an error creating the file 'logging':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'logging'
We'll now try updating this file instead of creating it

There was an error creating the file 'detect_device':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'detect_device'
We'll now try updating this file instead of creating it

There was an error creating the file 'blacklist':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'blacklist'
We'll now try updating this file instead of creating it

There was an error creating the file 'office_ip_list':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'office_ip_list'
We'll now try updating this file instead of creating it

There was an error creating the file 'ab_tests_deliver':
409 - Conflict
Message: Duplicate record
Detail: Duplicate vcl: 'ab_tests_deliver'
We'll now try updating this file instead of creating it

Yay, the file 'detect_edition' was updated successfully
Yay, the file 'set_country_cookie' was updated successfully
Yay, the file 'detect_device' was updated successfully
Yay, the file 'office_ip_list' was updated successfully
Yay, the file 'blacklist' was updated successfully
Yay, the file 'ab_tests_callback' was updated successfully
Yay, the file 'ab_tests_recv' was updated successfully
Yay, the file 'ab_tests_config' was updated successfully
Yay, the file 'main' was updated successfully
Yay, the file 'ab_tests_deliver' was updated successfully
Yay, the file 'logging' was updated successfully
```

## Build

I find using [Gox](https://github.com/mitchellh/gox) the simplest way to build multiple OS versions of a Golang application:

```bash
go get github.com/mitchellh/gox

gox -osarch="linux/amd64" -osarch="darwin/amd64" -osarch="windows/amd64" -output="lataa.{{.OS}}"

./lataa.darwin -h
```

## Environment Variables

The use of environment variables help to reduce the amount of flags required to get stuff done. For example, I always diff against a stage environment of Fastly and so I don't want to have to put in the same credentials all the time.

Below is a list of environment variables this tool supports:

* `FASTLY_API_TOKEN`
* `FASTLY_SERVICE_ID`
* `VCL_DIRECTORY`
* `VCL_MATCH_DIRECTORY`
* `VCL_SKIP_DIRECTORY`
