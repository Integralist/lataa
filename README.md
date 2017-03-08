# Lataa

Lataa is a cli tool, built in [Go](https://golang.org), used to upload local VCL files to a [Fastly CDN](https://www.fastly.com/) account.

> lataa is "upload" in Finnish

If you require a cli tool for diffing your local VCL files against a remote version within a Fastly account, then see [Ero](https://github.com/Integralist/ero)

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

None of the following snippets will work without one of the listed flags (although, any snippet that doesn't specify a `-service` or `-token` flag, does presume the equivalent environment variable has been set in its place):

* `-clone-version`
* `-upload-version`
* `-use-latest-version`

> See "[Logic Flow](#logic-flow)" below for details

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

View the status of a specific version (doesn't require additional flags):

```bash
lataa -get-version-status 123

Service 'abc' version '123' is 'not activated'
```

Here is that example again but when we specify an incorrect version:

```bash
lataa -get-version-status 999

There was a problem getting version 999

404 - Not Found
Message: Record not found
Detail: Couldn't find Version '[abc, 999, 0000-00-00 00:00:00]'
```

Activate a specific version (doesn't require additional flags):

```bash
lataa -activate-version 123 

Service 'abc' now has version '123' activated
```

## Logic Flow

The application has a specific set of flow controls it uses for determining what to do:

If user doesn't state what version to clone from (e.g. `-clone-version`), then we'll check to see if they specified an existing version to upload to instead (e.g. `-upload-version`). 

If they don't specify a specific version, then we'll presume they want to clone from the latest version. But if the user provides the `-use-latest-version` flag, then we wont clone from the latest but instead we will use the latest version for uploading to.

> Note: if the latest version is already activated, then we'll error appropriately

## Example

The following example execution has presumed the use of the environment variables: `VCL_DIRECTORY`, `FASTLY_API_TOKEN`, `FASTLY_SERVICE_ID` to keep the length of the command short.

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

Yay, the file 'detect_edition' was uploaded successfully
Yay, the file 'set_country_cookie' was uploaded successfully
Yay, the file 'detect_device' was uploaded successfully
Yay, the file 'office_ip_list' was uploaded successfully
Yay, the file 'blacklist' was uploaded successfully
Yay, the file 'ab_tests_callback' was uploaded successfully
Yay, the file 'ab_tests_recv' was uploaded successfully
Yay, the file 'ab_tests_config' was uploaded successfully
Yay, the file 'main' was uploaded successfully
Yay, the file 'ab_tests_deliver' was uploaded successfully
Yay, the file 'logging' was uploaded successfully
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
