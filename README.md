# Autoscan

Autoscan replaces the default Plex and Emby behaviour for picking up file changes on the file system.
Autoscan integrates with Sonarr, Radarr and Lidarr (with Google Drive coming soon!) to fetch changes in near real-time without relying on the file system.

Wait, what happened to [Plex Autoscan](https://github.com/l3uddz/plex_autoscan)?
Well, Autoscan is a rewrite of the original Plex Autoscan written in the Go language.
In addition, this rewrite introduces a more modular approach and should be easy to extend in the future.

## Table of contents

- [Early Access](#early-access)
  - [Installing autoscan](#installing-autoscan)
- [Introduction](#introduction)
  - [Rewriting paths](#rewriting-paths)
  - [Triggers](#triggers)
  - [Processor](#processor)
  - [Targets](#targets)
- [Other installation options](#other-installation-options)
  - [Docker](#docker)

## Early Access

**We are looking for technical writers! If you have ideas on how to improve Autoscan's documentation, please write [@m-rots](mailto:stormtimmermans@icloud.com?subject=Autoscan%20technical%20writer) an email.**

We have not finished all work on Autoscan yet, and are still working on some things.

The major feature which is currently MIA:

- Google Drive monitoring (Shared Drives exclusively)

Some small things we are still working on:

- Automating the testing of the processor's business logic
- Automating the testing of Emby
- Automating the testing of Plex

In addition, the code we currently do have is not yet finalised.
Certain files may get moved around a bit, internal APIs might change, etc.

However, we are proud of the rewrite and are eager to know your opinion!

### Installing autoscan

As Autoscan is still in active development, we highly recommend you to fetch the latest state of the master branch at all times.

To install the autoscan CLI on your system, make sure:

1. Your machine runs Linux, macOS or WSL2
2. You have [Go](https://golang.org/doc/install) installed (1.14 preferred)
3. You have a GCC compiler present \
  *Yup, we need to link to C because of SQLite >:(*
4. Clone this repository and cd into it from the terminal
5. Run `go build -o autoscan ./cmd/autoscan` from the terminal

You should now have a binary with the name `autoscan` in the root directory of the project.
To start autoscan, simply run `./autoscan`. If you want autoscan to be globally available, move it to `/bin` or `/usr/local/bin`.

If you need to debug certain Autoscan behaviour, either add the `-v` flag for debug mode or the `-vv` flag for trace mode to get even more details about internal behaviour.

We also offer a [Docker image](#docker)! However, its configuration may be a bit complex as it requires a good understanding of Autoscan's rewriting capabilities. We hope to provide detailed instructions on these rewriting capabilities in the near future!

## Introduction

Autoscan is split into three distinct modules:

- Triggers
- Processor
- Targets

### Rewriting paths

Triggers, targets and the processor all live in different contexts. Some are used in Docker containers, others run on host OS, it can be a big mess!

That's where rewrite rules come into play. They allow you to translate paths between a trigger's / target's perspective and the processor's perspective.

**Before you begin, make sure you understand how regular expressions work!** \
Make sure you know how capture groups work, as these are used for the `to` field.

Triggers can receive paths from any source: A remote server, a Docker container and the local file system. The `rewrite` field can be defined for each individual trigger. The `from` should be a regexp pattern describing the path from the trigger's perspective. The `to` should then convert this path into a path which is local to Autoscan.

Targets work the other way around. They have to convert the path local to Autoscan to a path understood by the target, which can be a Docker container, remote server, etc. The `from` should be a regexp pattern describing the path from Autoscan's perspective. The `to` should then convert this path into a path which is local to the target.

It is important that all three modules can have access to a file. When a trigger receives a scan, then the file should be available from both the processor and all targets.

#### Simple example

- Sonarr running in a Docker container (same example works for Lidarr and Radarr)
- Autoscan running on the host OS (not in a container)
- Plex running in a Docker container

The following config only defines rewrite paths, this should not be used directly!

```yaml
triggers:
  sonarr:
    - rewrite:
        # /tv contains folders with tv shows
        # This path is used within the Sonarr Docker container
        from: /tv/*

        # /mnt/unionfs/Media/TV links to the same folder, though from the host OS
        # This folder is accessed by Autoscan
        to: /mnt/unionfs/Media/TV/$1

targets:
  plex:
    - rewrite:
        # Same folder as above, accessible by Autoscan.
        # Note how we strip the "TV" part,
        # as we want both Movies and TV.
        from: /mnt/unionfs/Media/*

        # This path is used within the Plex Docker container
        to: /data/$1
```

Let's take a look at the journey of the path `/tv/Westworld/Season 1/s01e01.mkv` coming from Sonarr.

1. Sonarr's path is translated to a path local to Autoscan. \
  `/mnt/unionfs/Media/TV/Westworld/Season 1/s01e01.mkv`
2. The path is accessed by Autoscan to check whether it exists and adds it to the datastore.
3. Autoscan's path is translated to a path local to Plex. \
  `/data/TV/Season 1/s01e01.mkv`

This should be all that's needed to get you going. Good luck!

### Triggers

Triggers are the 'input' of Autoscan.
They translate incoming data into a common data format called the Scan.

We plan to support two kinds of triggers in GA:

- Daemon processes.
  These triggers run in the background and fetch resources based on a cron schedule or in real-time. \
  *Currently not available, but expected to arrive in GA.*

- Webhooks.
  These triggers expose HTTP handlers which can be added to the trigger's software. \
  *Available.*

Each trigger consists of at least:

- A unique identifier: think of Drive IDs and HTTP routes. \
  *Webhooks use /triggers/ + their name to uniquely identify themselves.*

- Trigger-wide priority: higher priorities are processed sooner. \
  *Defaults to 0.*

- RegExp-based rewriting rules: translate a path given by the trigger to a path on the local file system. \
  *If the paths are identical between the trigger and the local file system, then the `rewrite` field should be ignored.*

#### Webhooks

Webhooks, also known as HTTPTriggers internally, process HTTP requests on their exposed endpoints.
They should be tailor-made for the software they plan to support.

Each instance of a webhook exposes a route which is added to Autoscan's main router.

If one wants to configure a HTTPTrigger with multiple distinct configurations, then these configurations MUST provide a field called `Name` which uniquely identifies the trigger.
The name field is then used to create the route: `/triggers/:name`.

The following webhooks are currently provided by Autoscan:

- Sonarr
- Radarr
- Lidarr

#### Configuration

A snippet of the `config.yml` file showcasing what is possible.
You can mix and match exactly the way you like:

```yaml
# Optionally, protect your webhooks with authentication
authentication:
  username: hello there
  password: general kenobi

# port for Autoscan webhooks to listen on
port: 3030

triggers:
  sonarr:
    - name: sonarr-docker # /triggers/sonarr-docker
      priority: 2

      # Rewrite the path from within the container
      # to your local filesystem.
      rewrite:
        from: /tv/*
        to: /mnt/unionfs/Media/TV/$1

  radarr:
    - name: radarr   # /triggers/radarr
      priority: 2
    - name: radarr4k # /triggers/radarr4k
      priority: 5
  lidarr:
    - name: lidarr   # /triggers/lidarr
      priority: 1
```

#### Connecting the -arrs

To add your webhook to Sonarr, Radarr or Lidarr, do:

1. Open the `settings` page in Sonarr/Radarr/Lidarr
2. Select the tab `connect`
3. Click on the big plus sign
4. Select `webhook`
5. Use `Autoscan` as name (or whatever you prefer)
6. Select `On Import` and `On Upgrade`
7. Set the URL to Autoscan's URL and add `/triggers/:name` where name is the name set in the trigger's config.
8. Optional: set username and password.

### Processor

Triggers pass the Scans they receive to the processor.
The processor then saves the Scans to its datastore.

*The processor uses SQLite as its datastore, feel free to hack around!*

In a separate process, the processor selects Scans from the datastore.
It will always group files belonging to the same folder together and it waits until all the files in that folder are older than the `minimum-age`, which defaults to 5 minutes.

When all files are older than the minimum age, the processor will check whether all files exist on the local file system.
When at least one file exists on the file system, then the processor will call all the configured targets in parallel to request a folder scan.

When a file does not exist, the processor will increment the `retries` field of the Scan.
It also resets the timestamp so the file will not get scanned for at least `minimum-age`.
A Scan can only be retried up to a maximum number of retries, which defaults to 5.

#### Anchor files

To prevent the processor from calling targets when a remote mount is offline, you can define a list of so called `anchor files`.
These anchor files do not have any special properties and often have no content.
However, they can be used to check whether a file exists on the file system.
If the file does not exist and you have not made any changes to the file, then it is certain that the remote mount must be offline or the software is having problems.

When an anchor file is unavailable, the processor will halt its operations until the file is back online.

We suggest you to use different anchor file names if you merge multiple remote mounts together with a tool such as [UnionFS](https://unionfs.filesystems.org) or [MergerFS](https://github.com/trapexit/mergerfs).
Each remote mount MUST have its own anchor file and its own name for that anchor file.
In addition, make sure to define the 'merged' path to the file and not the remote mount path.
This helps check whether the union-software is working correctly as well.

#### Customising the processor

The processor allows you to set the maximum number of retries, as well as the minimum age of a Scan.
In addition, you can also define a list of anchor files.

A snippet of the `config.yml` file:

```yaml
# override the maximum number of retries
retries: 10

# override the minimum age to 2 minutes:
minimum-age: 2m

# set multiple anchor files
anchors:
  - /mnt/unionfs/drive1.anchor
  - /mnt/unionfs/drive2.anchor
```

The `minimum-age` field should be given a string in the following format:

- `1s` if the min-age should be set at 1 second.
- `5m` if the min-age should be set at 5 minutes.
- `1m30s` if the min-age should be set at 1 minute and 30 seconds.
- `1h` if the min-age should be set at 1 hour.

*Please do not forget the `s`, `m` or `h` suffix, otherwise the time unit defaults to nanoseconds.*

### Targets

While collecting Scans is fun and all, they need to have a final destination.
Targets are these final destinations and are given Scans from the processor, one batch at a time.

Autoscan currently supports two targets:

- Plex
- Emby

#### Plex

Autoscan replaces Plex's default behaviour of updating the Plex library automatically.
Therefore, it is advised to turn off Plex's `Update my library automatically` feature.

You can setup one or multiple Plex targets in the config:

```yaml
targets:
  plex:
    - url: https://plex.domain.tld # URL of your Plex server
      database: /opt/plex/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db # Path to the Plex database file
      token: XXXX # Plex API Token
      rewrite:
        from: /mnt/unionfs/Media/* # local file system
        to: /data/$1 # path accessible by the Plex docker container (if applicable)
```

There are a couple of things to take note of in the config:

- URL. The URL can link to the docker container directly, the localhost or a reverse proxy sitting in front of Plex.
- Database. Autoscan needs access to the database file to check whether files have actually been changed. The database file is named `com.plexapp.plugins.library.db` and you MUST provide a path to this file which can be accessed by Autoscan. \
  *An example path is given in the above config file.*
- Token. We need a Plex API Token to make requests on your behalf. [This article](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/) should help you out.
- Rewrite. If Plex is not running on the host OS, but in a Docker container (or Autoscan is running in a Docker container), then you need to rewrite paths accordingly. Check out our [rewriting section](#rewriting-paths) for more info.

#### Emby

While Emby provides much better behaviour out of the box than Plex, it still might be useful to use Autoscan for even better performance.

You can setup one or multiple Emby targets in the config:

```yaml
targets:
  emby:
    - url: https://emby.domain.tld # URL of your Emby server
      database: /opt/emby/data/library.db # Path to the Emby database file
      token: XXXX # Emby API Token
      rewrite:
        from: /mnt/unionfs/Media/* # local file system
        to: /data/$1 # path accessible by the Emby docker container (if applicable)
```

- URL. The URL can link to the docker container directly, the localhost or a reverse proxy sitting in front of Emby.
- Database. Autoscan needs access to the database file to check whether files have actually been changed. The database file is named `library.db` and you MUST provide a path to this file which can be accessed by Autoscan. \
  *An example path is given in the above config file.*
- Token. We need an Emby API Token to make requests on your behalf. [This article](https://github.com/MediaBrowser/Emby/wiki/Api-Key-Authentication) should help you out. \
  *It's a bit out of date, but I'm sure you will manage!*
- Rewrite. If Emby is not running on the host OS, but in a Docker container (or Autoscan is running in a Docker container), then you need to rewrite paths accordingly. Check out our [rewriting section](#rewriting-paths) for more info.

## Other installation options

### Docker

Autoscan has an accompanying docker image which can be found on [Docker Hub](https://hub.docker.com/repository/docker/cloudb0x/autoscan).

Autoscan requires access to the following files:

- All files being passed between the triggers and the targets. \
  *Just mount the source directory, for many people this is `/mnt/unionfs`.*
- The Plex database (make sure you mount the folder, not the db file directly). \
  *Only when using Plex as a target.*
- The Emby database (make sure you mount the folder, not the db file directly). \
  *Only when using Emby as a target.*

Make sure these files are available within the Autoscan container.
Remember that you MUST use [rewriting rules](#rewriting-paths) if paths are not identical between triggers, autoscan and targets. These rules can be set from the config for each trigger and target individually.

#### Version Tags

Autoscan's Docker image provides various versions that are available via tags. The `latest` tag usually provides the latest stable version. Others are considered under development and caution must be exercised when using them.

| Tag | Description |
| :----: | --- |
| latest | Latest stable version from a tagged GitHub release |
| master | Most recent GitHub master commit |

#### Usage

```bash
docker run \
  --name=autoscan \
  -e "PUID=1000" \
  -e "PGID=1000" \
  -p 3030:3030 \
  -v "/opt/autoscan:/config" \
  -v "/mnt/unionfs:/mnt/unionfs:ro" \
  -v "/opt/plex:/data/plex:ro" \
  -v "/opt/emby:/data/emby:ro" \
  --restart=unless-stopped \
  -d cloudb0x/autoscan
```

#### Parameters

Autoscan's Docker image supports the following parameters.

| Parameter | Function |
| :----: | --- |
| `-p 3030:3030` | The port used by Autoscan's webhook triggers |
| `-e PUID=1000` | The UserID to run the Autoscan binary as |
| `-e PGID=1000` | The GroupID to run the Autoscan binary as |
| `-e AUTOSCAN_VERBOSITY=0` | The Autoscan logging verbosity level to use. (0 = info, 1 = debug, 2 = trace) |
| `-v /config` | Autoscan's config and database file |

Any other volumes can be referenced within Autoscan's config file `config.yml`, assuming it has been specified as a volume.

#### Cloudbox

The following Docker setup should work for many Cloudbox users.

**WARNING: You still need to configure the `config.yml` file!**

Make sure to replace `DOMAIN.TLD` with your domain and `YOUR_EMAIL` with your email.

```bash
docker run \
  --name=autoscan \
  -e "PUID=1000" \
  -e "PGID=1001" \
  -e "VIRTUAL_HOST=autoscan.DOMAIN.TLD" \
  -e "VIRTUAL_PORT=3030" \
  -e "LETSENCRYPT_HOST=autoscan.DOMAIN.TLD" \
  -e "LETSENCRYPT_EMAIL=YOUR_EMAIL" \
  -v "/opt/autoscan:/config" \
  -v "/opt/plex:/data/plex:ro" \
  -v "/opt/emby:/data/emby:ro" \
  -v "/mnt:/mnt:ro" \
  --label="com.github.cloudbox.cloudbox_managed=true" \
  --network=cloudbox \
  --network-alias=autoscan  \
  --restart=unless-stopped \
  -d cloudb0x/autoscan
```
