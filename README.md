<p align="center">
  <a href="https://yarn.social/">
    <img alt="Yarn social" src="https://git.mills.io/yarnsocial/assets/raw/branch/master/yarn.svg" width="220"/>
  </a>
</p>
<h1 align="center">Yarn - a decentralised self-hosted social media that has a privacy-first focus.</h1>

<p align="center">
  <a href="https://goreportcard.com/report/git.mills.io/yarnsocial/yarn" title="Go Report Card">
    <img src="https://goreportcard.com/badge/git.mills.io/yarnsocial/yarn">
  </a>
  <a href="https://pkg.go.dev/git.mills.io/yarnsocial/yarn" title="GoDoc">
    <img src="https://pkg.go.dev/badge/git.mills.io/yarnsocial/yarn.svg">
  </a>
  <a href="https://opensource.org/licenses/AGPLv3" title="License: AGPLv3">
    <img src="https://img.shields.io/badge/License-AGPLv3-blue.svg">
  </a>
  <a href="https://hub.docker.com/u/prologic/yarnd" title="Docker Pulls">
    <img src="https://img.shields.io/docker/pulls/prologic/yarnd">
  </a>
</p>

<p align="center">
  <a href="/yarnsocial/yarn/src/branch/main/README_ZH.md">View the chinese version of this document</a>
</p>

Table of Contents:
<!--toc-->
- [Quick Start](#quick-start)
- [Installation](#installation)
  - [Pre-built Binaries](#pre-built-binaries)
  - [Using Homebrew](#using-homebrew)
  - [Building from source](#building-from-source)
- [Clients](#clients)
  - [Command-line Client](#command-line-client)
  - [Other clients](#other-clients)
- [Deployment](#deployment)
  - [Deploy With Docker](#deploy-with-docker)
  - [Deploy with Docker Swarm](#deploy-with-docker-swarm)
  - [Other deployments](#other-deployments)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [Contributors](#contributors)
- [Related Projects](#related-projects)
- [License](#license)
<!-- tocstop -->

## Quick Start

The quickest and easiest way to try our Yarn.social is to try the [demo](demo):

https://demo.yarn.social/

> [!WARNING]  
> This demo instance is deliberately kept locked down to prevent abuse.
> Some functionality will be restricted as a result.

Alternatively you can try things out locally by running:

```console
docker run -p 8000:8000 prologic/yarnd
```

And visit http://localhost:8000 in your browser.

See the [Deployment](#deployment) section for other detailed deployment options.

## Installation

### Pre-built Binaries

As a first point, please try to use one of the pre-built binaries that are
available on the [Releases](https://git.mills.io/yarnsocial/yarn/releases) page.

### Using Homebrew

We provide [Homebrew](https://brew.sh) formulae for macOS users for both the
command-line client (`yarnc`) as well as the server (`yarnd`).

```console
brew tap yarnsocial/yarn https://git.mills.io/yarnsocial/homebrew-yarn.git
brew install yarn
```

Run the server:

```console
yarnd
```

Run the command-line client:

```console
yarnc
```

### Building from source

This is an option if you are familiar with [Go](https://golang.org) development.

> [!IMPORTANT]
> Be sure to have `$GOBIN` (_if not empty_) or your `$GOPATH/bin`
> in your `$PATH`. See [Compile and install packages and dependencies](https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies)

1. Clone this repository (_this is important_)

```console
git clone https://git.mills.io/yarnsocial/yarn.git
```

2. Install required dependencies (_this is important_)

Linux, macOS:

```console
make deps
```

> [!NOTE]
> In order to get the media upload functions to work, you need to
> install ffmpeg and its associated `-dev` packages. Consult your OS software
> or package manager availability and how to install these dependencies.

FreeBSD:

- Install `gmake`
- Install `pkgconf` that brings `pkg-config`

```console
gmake deps
```

3. Build the binaries

Linux, macOS:

The server:

```console
make server
```

The client:

```console
make cli
```

List all options:

```console
make help
```

FreeBSD:

```console
gmake
```

## Clients

### Command-line Client

Every Yarn.social pod provides an API that can be used alongside the builtin
web interface. There is also a command-lint client that uses the API. Here's
a basic guide on using the command-line client:

1. Login to  your [Yarn.social](https://yarn.social) pod:

```#!console
$ ./yarnc login
INFO[0000] Using config file: /Users/prologic/.twt.yaml
Username:
```

2. Viewing your timeline

```#!console
$ ./yarnc timeline
INFO[0000] Using config file: /Users/prologic/.twt.yaml
> prologic (50 minutes ago)
Hey @rosaelefanten 👋 Nice to see you have a Twtxt feed! Saw your [Tweet](https://twitter.com/koehr_in/status/1326914925348982784?s=20) (_or at least I assume it was yours?_). Never heard of `aria2c` till now! 🤣 TIL

> dilbert (2 hours ago)
Angry Techn Writers ‣ https://dilbert.com/strip/2020-11-14
```

3. Making a Twt (_post_):

```#!console
$ ./yarnc post
INFO[0000] Using config file: /Users/prologic/.twt.yaml
Testing `yarn` the command-line client
INFO[0015] posting twt...
INFO[0016] post successful
```

For additional help on using the `yarnc` command-line client:

```#!console
$ yarnc help
This is the command-line client for Yarn.social pods running
yarnd. This tool allows a user to interact with a pod to view their timeline,
following feeds, make posts and managing their account.

Usage:
  yarnc [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  help        Help about any command
  login       Login and authenticate to a Yarn.social pod
  post        Post a new twt to a Yarn.social pod
  stats       Parses and performs statistical analytis on a Twtxt feed given a URL or local file
  timeline    Display your timeline

Flags:
  -c, --config string   set a custom config file (default "/Users/prologic/.yarnc.yml")
  -D, --debug           Enable debug logging
  -h, --help            help for yarnc
  -t, --token string    yarnd API token to use to authenticate to endpoints (default "$YARNC_TOKEN")
  -U, --uri string      yarnd API endpoint URI to connect to (default "http://localhost:8000/api/v1/")

Use "yarnc [command] --help" for more information about a command.
```

### Other clients

- [go.yarn.social/client](https://git.mills.io/yarnsocial/go-client)
  The officially maintained client library for accessing the `yarnd` API.
  This is used by the command-line client above.
- [twt.js](https://git.mills.io/yarnsocial/twt.js)
  An old unmaintained Javascript client. If you are interested in maintaining
  this client, please contact us. See [Contributing](#contributing)

## Deployment

### Deploy With Docker

If you are comfortable using [Docker](https://www.docker.com) you can easily
deploy a Yarn.social pod by using the provided Docker image
[prologic/yarnd](https://hub.docker.com/r/prologic/yarnd) which are regularly
published for both AMD64 and ARM64 platforms.

Running or testing `yarnd` locally is as easy as running:

```console
$ docker run -p 8000:8000 prologic/yarnd
```

And visiting http://localhost:8000 in your browser.

Alternatively you can also use [Docker Compose](https://docs.docker.com/compose/)
to manage your deployment. A sample `docker-compose.yaml` is provided in the
root directory of the source code. You can run it locally by running:

```console
docker-compose up -d
```

> [!NOTE]  
> The [Dockerfile](/Dockerfile) specifies that the container run as
> the user `yarnd` with `uid=1000`. Be sure that any volume(s) you
> mount into your container and use as the data storage (`-d/--data`)
> path and database storage path (`-s/--store`) is correctly configured
> to have the correct user/group ownership. e.g: `chorn -R 1000:1000 /data`

### Deploy with Docker Swarm

You can deploy `yarnd` to a [Docker Swarm](https://docs.docker.com/engine/swarm/)
cluster by utilising the provided `yarn.yaml` Docker Stack. This also depends on
and uses the [Traefik](https://docs.traefik.io/) ingress load balancer so you must
also have that configured and running in your cluster appropriately.

```console
docker stack deploy -c yarn.yml
```

See [deployment](./deployment/) for a full guide on a production deployment.

### Other deployments

To deploy a pod running `yarnd` on your system, install the `yarnd` binary in an
appropriate location such as `/usr/local/bin`. Next ensure you have a path for
storing your pod's data. This path will contain the database, feeds, avatars,
cache, index, and a few other files.

> [!IMPORTANT]  
> It is important that you backup your `/path/to/data` directory used for your
> pod regularly. Using tools like `tar` and `scp` is enough to copy and backup
> the files.

Run yarnd:

```console
$ ./yarnd -R
```

> [!NOTE]  
> Registrations are disabled by default so hence the `-R` flag above.

Then visit: http://localhost:8000/

You can configure other options by specifying them on the command-line
or via environment variables.

To view the available options simply run:

```console
./yarnd --help
```

Valid environment value names are the long-option version of a flag in all
uppercase with dashes replaced by an underscore `_`.

## Configuration

At a bare minimum you should set the following options:

- `-d /path/to/data`
- `-s bitcask:///path/to/data/twtxt.db` (_we will likely simplify/default this_)
- `-n <name>` to give your pod a unique name.
- `-u <url>` the base url (_public facing_) of how your pod will be reached on the web.
- `-R` to enable open registrations.
- `-O` to enable open profiles.

Most other configuration values _should_ be done via environment variables.

It is _recommended_ you pick an account you want to use to "administer" the
pod with and set the following environment values:

- `ADMIN_USER=username`
- `ADMIN_EMAIL=email`

In order to configure email settings for password recovery and the `/support`
and `/abuse` endpoints, you should set appropriate `SMTP_` values.

A production pod (_that is one run without the `-D/--debug` flag_) **MUST** be
configured with the following additional options:

- `API_SIGNING_KEY`
- `COOKIE_SECRET`
- `MAGICLINK_SECRET`

These values _should_ be generated with a secure random number generator and
be of length `64` characters long. You can use the following shell snippet
to generate secrets for your pod for the above values:

```console
$ cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1
```

There is a shell script in `./tools/gen-secrets.sh` you can use to conveniently generate the required secrets for a production pod. The output is designed to by copy/pasted into a `docker-compose.yml` file with the right indentation.

> [!CAUTION]
> **DO NOT** publish or share these values.
> **BE SURE** to only set them as env vars.

## Contributing

Interested in contributing to this project? You are welcome! Here are some ways
you can contribute:

- [File an Issue](https://git.mills.io/yarnsocial/yarn/issues/new) -- For a bug,
  or interesting idea you have for a new feature or just general questions.
- Submit a Pull-Request or two! We welcome all PR(s) that improve the project!

Please see the [Contributing Guidelines](/CONTRIBUTING.md) and checkout the
[Developer Documentation](https://dev.twtxt.net) or over at [/docs](/docs).

## Contributors

Thank you to all those that have contributed to this project, battle-tested it, used it in their own projects or products, fixed bugs, improved performance and even fix tiny typos in documentation! Thank you and keep contributing!

You can find an [AUTHORS](/AUTHORS) file where we keep a list of contributors to the project. If you contribute a PR please consider adding your name there.

## Related Projects

- [Yarn.social](https://git.mills.io/yarnsocial/yarn.social) -- [Yarn.social](https://yarn.social) landing page
- [Search](https://git.mills.io/yarnsocial/yarns) -- The [Yarn.social](https://yarn.social) search engine hosted at [search.twtxt.net](https://search.twtxt.net)
- [App](https://git.mills.io/yarnsocial/app) -- Our Flutter iOS and Android Mobile App
- [Feeds](https://git.mills.io/yarnsocial/feeds) -- RSS/Atom/Twitter to [Twtxt](https://twtxt.readthedocs.org) aggregator service hosted at [feeds.twtxt.net](https://feeds.twtxt.net)

## License

`yarn` is licensed under the terms of the [AGPLv3 License](/LICENSE)
