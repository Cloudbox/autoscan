# Autoscan

Autoscan replaces the default Plex and Emby behaviour for picking up file changes on the file system.
Autoscan integrates with Sonarr, Radarr and Lidarr (with Google Drive coming soon!) to fetch changes in near real-time without relying on the file system.

Wait, what happened to [Plex Autoscan](https://github.com/l3uddz/plex_autoscan)?
Well, Autoscan is a rewrite of the original Plex Autoscan written in the Go language.
In addition, this rewrite introduces a more modular approach and should be easy to extend in the future.

## Early Access

We have not finished all work on Autoscan yet, and are still working on some things.

The major feature which is currently MIA:

- Google Drive monitoring

Some small things we are still working on:

- Automating the testing of the processor's business logic
- Automating the testing of Emby
- Refactoring the Plex tests

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

We also offer a [Docker image](https://hub.docker.com/r/cloudb0x/autoscan)! However, its configuration may be a bit complex as it requires a good understanding of Autoscan's rewriting capabilities. We hope to provide detailed instructions on these rewriting capabilities in the near future!

## Introduction

Autoscan is split into three distinct modules:

- Triggers
- Processor
- Targets

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
auth:
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
