Autoscan, [A-Train](https://github.com/m-rots/a-train) and [Bernard](https://github.com/m-rots/bernard-rs) are no longer actively maintained. All projects are considered feature frozen and when compatibility with Google Drive, Plex, Emby, Jellyfin and the -arrs inevitably breaks, no fixes will be provided and such an event will officially mark these projects as end of life.
As all three projects have permissible open source licenses, feel free to start a fork and continue development. Ownership of these repositories as well as the Docker images will not be transferred.

# Autoscan

Autoscan replaces the default Plex and Emby behaviour for picking up file changes on the file system.
Autoscan integrates with Sonarr, Radarr, Readarr, Lidarr and Google Drive to fetch changes in near real-time without relying on the file system.

Wait, what happened to [Plex Autoscan](https://github.com/l3uddz/plex_autoscan)?
Well, Autoscan is a rewrite of the original Plex Autoscan written in the Go language.
In addition, this rewrite introduces a more modular approach and should be easy to extend in the future.

## Comparison to Plex Autoscan

- [A-Train](https://github.com/m-rots/a-train/pkgs/container/a-train), Autoscan's Google Drive integration, only supports Shared Drives and requires Service Account authentication.
- A-Train does not support RClone Crypt remotes.
- Autoscan does not rely on manual trash deletion when connected to Plex. Therefore, you should re-enable the `Empty trash automatically after every scan` setting in Plex.

Autoscan also improves upon [Plex Autoscan](https://github.com/l3uddz/plex_autoscan) by adding the following features:

- Autoscan supports Plex music libraries.
- Autoscan adds additional support for Emby and Jellyfin.
- Autoscan can send _scans_ to multiple Plex, Emby and Jellyfin servers.

## Installing autoscan

Autoscan offers [pre-compiled binaries](https://github.com/Cloudbox/autoscan/releases/latest) for both Linux and MacOS for each official release. In addition, we also offer a [Docker image](#docker)!

Alternatively, you can build the Autoscan binary yourself.
To build the autoscan CLI on your system, make sure:

1. Your machine runs Linux, macOS or WSL2
2. You have [Go](https://golang.org/doc/install) installed (1.16 or later)
3. Clone this repository and cd into it from the terminal
4. Run `go build -o autoscan ./cmd/autoscan` from the terminal

You should now have a binary with the name `autoscan` in the root directory of the project.
To start autoscan, simply run `./autoscan`. If you want autoscan to be globally available, move it to `/bin` or `/usr/local/bin`.

If you need to debug certain Autoscan behaviour, either add the `-v` flag for debug mode or the `-vv` flag for trace mode to get even more details about internal behaviour.

## Overview

Autoscan is split into three distinct modules:

- Triggers
- Processor
- Targets

### Rewriting paths

Triggers, targets and the processor all live in different contexts. Some are used in Docker containers, others run on host OS, it can be a big mess!

That's where rewrite rules come into play. They allow you to translate paths between a trigger's / target's perspective and the processor's perspective.

**Before you begin, make sure you understand how regular expressions work!** \
Make sure you know how capture groups work, as these are used for the `to` field.

Triggers can receive paths from any source: A remote server, a Docker container and the local file system. The `rewrite` field can be defined for each individual trigger. This field can contain multiple rewriting rules. Therefore, each rule should have a `-` to indicate the next rule on the list. The `from` should be a regexp pattern describing the path from the trigger's perspective. The `to` should then convert this path into a path which is local to Autoscan.

Targets work the other way around. They have to convert the path local to Autoscan to a path understood by the target, which can be a Docker container, remote server, etc. The `from` should be a regexp pattern describing the path from Autoscan's perspective. The `to` should then convert this path into a path which is local to the target.

It is important that all three modules can have access to a file. When a trigger receives a scan, then the file should be available from both the processor and all targets.

#### Simple example

- Sonarr running in a Docker container (same example works for Lidarr, Radarr and Readarr)
- Autoscan running on the host OS (not in a container)
- Plex running in a Docker container

The following config only defines rewrite paths, this should not be used directly!

```yaml
triggers:
  sonarr:
    - rewrite:
          # /tv contains folders with tv shows
          # This path is used within the Sonarr Docker container
        - from: /tv/

          # /mnt/unionfs/Media/TV links to the same folder, though from the host OS
          # This folder is accessed by Autoscan
          to: /mnt/unionfs/Media/TV/

targets:
  plex:
    - rewrite:
          # Same folder as above, accessible by Autoscan.
          # Note how we strip the "TV" part,
          # as we want both Movies and TV.
        - from: /mnt/unionfs/Media/

          # This path is used within the Plex Docker container
          to: /data/
```

Let's take a look at the journey of the path `/tv/Westworld/Season 1/s01e01.mkv` coming from Sonarr.

1. Sonarr's path is translated to a path local to Autoscan. \
  `/mnt/unionfs/Media/TV/Westworld/Season 1/s01e01.mkv`
2. The path is accessed by Autoscan to check whether it exists and adds it to the datastore.
3. Autoscan's path is translated to a path local to Plex. \
  `/data/TV/Season 1/s01e01.mkv`

This should be all that's needed to get you going. Good luck!

## Triggers

Triggers are the 'input' of Autoscan.
They translate incoming data into a common data format called the Scan.

Autoscan currently supports the following triggers:

- [A-Train](https://github.com/m-rots/a-train/pkgs/container/a-train): The official Google Drive trigger for Autoscan. \
  _A-Train is [available separately](https://github.com/m-rots/a-train/pkgs/container/a-train)._

- Inotify: Listens for changes on the file system. \
  **This should not be used on top of RClone mounts.** \
  *Bugs may still exist.*

- Manual: When you want to scan a path manually.

- The -arrs: Lidarr, Sonarr, Radarr and Readarr. \
  Webhook support for Lidarr, Sonarr, Radarr and Readarr.

All triggers support:

- Trigger-wide priority: higher priorities are processed sooner. \
  *Defaults to 0.*

- RegExp-based rewriting rules: translate a path given by the trigger to a path on the local file system. \
  *If the paths are identical between the trigger and the local file system, then the `rewrite` field should be ignored.*

### A-Train

Autoscan can monitor Google Drive through [A-Train](https://github.com/m-rots/a-train/pkgs/container/a-train). A-Train is a stand-alone tool created by the Autoscan developers and is officially part of the Autoscan project.

The A-Train trigger configuration is not required, as Autoscan automatically listens for A-Train requests. However, to configure global and drive-specific rewrite rules, you could add A-Train to your config:

```yaml
triggers:
  a-train:
    priority: 5
    rewrite: # Global rewrites
      - from: ^/Media/
        to: /mnt/unionfs/Media/
    # Drives only need to be given when Drive-specific rewrites are used
    drives:
      - id: 0A1xxxxxxxxxUk9PVA # The ID of Shared Drive #1
        rewrite: # Drive-specific rewrite (has priority over global rewrite)
          - from: ^/TV/
            to: /mnt/unionfs/TV/
      - id: 0A2xxxxxxxxxUk9PVA # The ID of Shared Drive #2
        rewrite: # Drive-specific rewrite (has priority over global rewrite)
          - from: ^/Movies/
            to: /mnt/unionfs/Movies/
```

### Manual

**Note: You can visit `/triggers/manual` within a browser to manually submit requests**

Autoscan also supports a `manual` webhook for custom scripts or for software which is not supported by Autoscan directly. The manual endpoint is available at `/triggers/manual`.

The manual endpoint accepts one or multiple directory paths as input and should be given one or multiple `dir` query parameters. Just like the other webhooks, the manual webhook is protected with basic authentication if the `auth` option is set in the config file of the user.

URL template: `POST /triggers/manual?dir=$path1&dir=$path2`

The following curl command sends a request to Autoscan to scan the directories `/test/one` and `/test/two`:

```bash
curl --request POST \
  --url 'http://localhost:3030/triggers/manual?dir=%2Ftest%2Fone&dir=%2Ftest%2Ftwo' \
  --header 'Authorization: Basic aGVsbG8gdGhlcmU6Z2VuZXJhbCBrZW5vYmk='
```

### The -arrs

If one wants to configure a HTTPTrigger with multiple distinct configurations, then these configurations MUST provide a field called `Name` which uniquely identifies the trigger.
The name field is then used to create the route: `/triggers/:name`.

The following -arrs are currently provided by Autoscan:

- Lidarr
- Radarr
- Readarr
- Sonarr

#### Connecting the -arrs

To add your webhook to Sonarr, Radarr, Readarr or Lidarr, do:

1. Open the `settings` page in Sonarr/Radarr/Readarr/Lidarr
2. Select the tab `connect`
3. Click on the big plus sign
4. Select `webhook`
5. Use `Autoscan` as name (or whatever you prefer)
6. Select `On Import` and `On Upgrade`
7. Set the URL to Autoscan's URL and add `/triggers/:name` where name is the name set in the trigger's config.
8. Optional: set username and password.

#### The latest events

Autoscan also supports the following events in the latest versions of Radarr and Sonarr:
- `Rename`
- `On Movie Delete` and `On Series Delete`
- `On Movie File Delete` and `On Episode File Delete`

We are not 100% sure whether these three events cover all the possible file system interactions.
So for now, please do keep using Bernard or the Inotify trigger to fetch all scans.

### Configuration

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
  # The manual trigger is always enabled, the config only adjusts its priority and the rewrite rules.
  manual:
    priority: 5
    rewrite:
      - from: ^/Media/
        to: /mnt/unionfs/Media/

  a-train:
    priority: 5
    rewrite: # Global rewrites
      - from: ^/Media/
        to: /mnt/unionfs/Media/
    # Drives only need to be given when Drive-specific rewrites are used
    drives:
      - id: 0A1xxxxxxxxxUk9PVA # The ID of Shared Drive #1
        rewrite: # Drive-specific rewrite (has priority over global rewrite)
          - from: ^/TV/
            to: /mnt/unionfs/TV/

  inotify:
    - priority: 0

      # filter with regular expressions
      include:
        - ^/mnt/unionfs/Media/
      exclude:
        - '\.(srt|pdf)$'

      # rewrite inotify path to unified filesystem
      rewrite:
            - from: ^/mnt/local/Media/
              to: /mnt/unionfs/Media/

      # local filesystem paths to monitor
      paths:
        - path: /mnt/local/Media

  lidarr:
    - name: lidarr   # /triggers/lidarr
      priority: 1

  radarr:
    - name: radarr   # /triggers/radarr
      priority: 2
    - name: radarr4k # /triggers/radarr4k
      priority: 5

  readarr:
    - name: readarr  # /triggers/readarr
      priority: 1

  sonarr:
    - name: sonarr-docker # /triggers/sonarr-docker
      priority: 2

      # Rewrite the path from within the container
      # to your local filesystem.
      rewrite:
        - from: /tv/
          to: /mnt/unionfs/Media/TV/
```

## Processor

Triggers pass the Scans they receive to the processor.
The processor then saves the Scans to its datastore.

*The processor uses SQLite as its datastore, feel free to hack around!*

In a separate process, the processor selects Scans from the datastore.
It will always group files belonging to the same folder together and it waits until all the files in that folder are older than the `minimum-age`, which defaults to 10 minutes.

When all files are older than the minimum age, then the processor will call all the configured targets in parallel to request a folder scan.

### Anchor files

To prevent the processor from calling targets when a remote mount is offline, you can define a list of so called `anchor files`.
These anchor files do not have any special properties and often have no content.
However, they can be used to check whether a file exists on the file system.
If the file does not exist and you have not made any changes to the file, then it is certain that the remote mount must be offline or the software is having problems.

When an anchor file is unavailable, the processor will halt its operations until the file is back online.

We suggest you to use different anchor file names if you merge multiple remote mounts together with a tool such as [UnionFS](https://unionfs.filesystems.org) or [MergerFS](https://github.com/trapexit/mergerfs).
Each remote mount MUST have its own anchor file and its own name for that anchor file.
In addition, make sure to define the 'merged' path to the file and not the remote mount path.
This helps check whether the union-software is working correctly as well.

### Minimum age

Autoscan does not check whether scan requests received by triggers exist on the file system.
Therefore, to make sure a file exists before it reaches the targets, you should set a minimum age.
The minimum age delays the scan from being send to the targets after it has been added to the queue by a trigger.
The default minimum age is set at 10 minutes to prevent common synchronisation issues.

### Customising the processor

The processor allows you to set the minimum age of a Scan.
In addition, you can also define a list of anchor files.

A snippet of the `config.yml` file:

```yaml
# override the minimum age to 30 minutes:
minimum-age: 30m

# override the delay between processed scans:
# defaults to 5 seconds
scan-delay: 15s

# override the interval scan stats are displayed:
# defaults to 1 hour / 0s to disable
scan-stats: 1m

# set multiple anchor files
anchors:
  - /mnt/unionfs/drive1.anchor
  - /mnt/unionfs/drive2.anchor
```

The `minimum-age`, `scan-delay` and `scan-stats` fields should be given a string in the following format:

- `1s` if the min-age should be set at 1 second.
- `5m` if the min-age should be set at 5 minutes.
- `1m30s` if the min-age should be set at 1 minute and 30 seconds.
- `1h` if the min-age should be set at 1 hour.

*Please do not forget the `s`, `m` or `h` suffix, otherwise the time unit defaults to nanoseconds.*

Scan stats will print the following information at a configured interval:

- Scans processed
- Scans remaining

## Targets

While collecting Scans is fun and all, they need to have a final destination.
Targets are these final destinations and are given Scans from the processor, one batch at a time.

Autoscan currently supports the following targets:

- Plex
- Emby
- Jellyfin
- Autoscan

### Plex

Autoscan replaces Plex's default behaviour of updating the Plex library automatically.
Therefore, it is advised to turn off Plex's `Update my library automatically` feature.

You can setup one or multiple Plex targets in the config:

```yaml
targets:
  plex:
    - url: https://plex.domain.tld # URL of your Plex server
      token: XXXX # Plex API Token
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /data/ # path accessible by the Plex docker container (if applicable)
```

There are a couple of things to take note of in the config:

- URL. The URL can link to the docker container directly, the localhost or a reverse proxy sitting in front of Plex.
- Token. We need a Plex API Token to make requests on your behalf. [This article](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/) should help you out.
- Rewrite. If Plex is not running on the host OS, but in a Docker container (or Autoscan is running in a Docker container), then you need to rewrite paths accordingly. Check out our [rewriting section](#rewriting-paths) for more info.

### Emby

While Emby provides much better behaviour out of the box than Plex, it still might be useful to use Autoscan for even better performance.

You can setup one or multiple Emby targets in the config:

```yaml
targets:
  emby:
    - url: https://emby.domain.tld # URL of your Emby server
      token: XXXX # Emby API Token
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /data/ # path accessible by the Emby docker container (if applicable)
```

- URL. The URL can link to the docker container directly, the localhost or a reverse proxy sitting in front of Emby.
- Token. We need an Emby API Token to make requests on your behalf. [This article](https://github.com/MediaBrowser/Emby/wiki/Api-Key-Authentication) should help you out. \
  *It's a bit out of date, but I'm sure you will manage!*
- Rewrite. If Emby is not running on the host OS, but in a Docker container (or Autoscan is running in a Docker container), then you need to rewrite paths accordingly. Check out our [rewriting section](#rewriting-paths) for more info.

### Jellyfin

While Jellyfin provides much better behaviour out of the box than Plex, it still might be useful to use Autoscan for even better performance.

You can setup one or multiple Jellyfin targets in the config:

```yaml
targets:
  jellyfin:
    - url: https://jellyfin.domain.tld # URL of your Jellyfin server
      token: XXXX # Jellyfin API Token
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /data/ # path accessible by the Jellyfin docker container (if applicable)
```

- URL. The URL can link to the docker container directly, the localhost or a reverse proxy sitting in front of Jellyfin.
- Token. We need a Jellyfin API Token to make requests on your behalf. [This article](https://github.com/MediaBrowser/Emby/wiki/Api-Key-Authentication) should help you out. \
  *It's a bit out of date, but I'm sure you will manage!*
- Rewrite. If Jellyfin is not running on the host OS, but in a Docker container (or Autoscan is running in a Docker container), then you need to rewrite paths accordingly. Check out our [rewriting section](#rewriting-paths) for more info.

### Autoscan

You can also send scan requests to other instances of autoscan!

```yaml
targets:
  autoscan:
    - url: https://autoscan.domain.tld # URL of Autoscan
      username: XXXX # Username for remote autoscan instance
      password: XXXX # Password for remote autoscan instance
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /mnt/nfs/Media/ # path accessible by the remote autoscan instance (if applicable)
```

## Full config file

With the examples given in the [triggers](#triggers), [processor](#processor) and [targets](#targets) sections, here is what your full config file *could* look like:

```yaml
# <- processor ->

# override the minimum age to 30 minutes:
minimum-age: 30m

# set multiple anchor files
anchors:
  - /mnt/unionfs/drive1.anchor
  - /mnt/unionfs/drive2.anchor

# <- triggers ->

# Protect your webhooks with authentication
authentication:
  username: hello there
  password: general kenobi

# port for Autoscan webhooks to listen on
port: 3030

triggers:
  lidarr:
    - name: lidarr   # /triggers/lidarr
      priority: 1

  radarr:
    - name: radarr   # /triggers/radarr
      priority: 2
    - name: radarr4k # /triggers/radarr4k
      priority: 5

  readarr:
    - name: readarr  # /triggers/readarr
      priority: 1

  sonarr:
    - name: sonarr-docker # /triggers/sonarr-docker
      priority: 2

      # Rewrite the path from within the container
      # to your local filesystem.
      rewrite:
        - from: /tv/
          to: /mnt/unionfs/Media/TV/

# <- targets ->

targets:
  plex:
    - url: https://plex.domain.tld # URL of your Plex server
      token: XXXX # Plex API Token
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /data/ # path accessible by the Plex docker container (if applicable)

  emby:
    - url: https://emby.domain.tld # URL of your Emby server
      token: XXXX # Emby API Token
      rewrite:
        - from: /mnt/unionfs/Media/ # local file system
          to: /data/ # path accessible by the Emby docker container (if applicable)
```

## Other configuration options

```yaml
# Specify the port to listen on (3030 is the default when not specified)
port: 3030
```

```yaml
# Specify the host interface(s) to listen on (0.0.0.0 is the default when not specified)
host:
  - 127.0.0.1
  - 172.19.185.13
  - 192.168.0.1:5959
```

- If no port is specified, it will use the default port configured.
- This configuration option is only needed if you have a requirement to listen to multiple interfaces.

## Other installation options

### Docker

Autoscan has an accompanying docker image which can be found on [Docker Hub](https://hub.docker.com/r/cloudb0x/autoscan).

Autoscan requires access to all files being passed between the triggers and the targets. \
*Just mount the source directory, for many people this is `/mnt/unionfs`.*

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
  -e "PGID=1001" \
  -p 3030:3030 \
  -v "/opt/autoscan:/config" \
  -v "/mnt/unionfs:/mnt/unionfs:ro" \
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
  -v "/mnt:/mnt:ro" \
  --label="com.github.cloudbox.cloudbox_managed=true" \
  --network=cloudbox \
  --network-alias=autoscan  \
  --restart=unless-stopped \
  -d cloudb0x/autoscan
```
