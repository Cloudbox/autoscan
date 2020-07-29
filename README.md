## Introduction

Autoscan is split into three distinct modules:

- Triggers
- Processor
- Targets

### Triggers

Triggers are the 'input' of Autoscan.
They translate incoming data into a common data format called the Scan.

We plan to support two kinds of triggers in General Availability (GA):

- Daemon processes.
  These triggers run in the background and fetch resources based on a cron schedule. \
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
  *If the paths are the identical between the trigger and the local file system, then the `rewrite` field can be ignored.*

#### Webhooks

Webhooks, also known as HTTPTriggers internally, process HTTP requests on their exposed endpoints.
They should be tailor-made for the software they plan to support.

Each instance of a webhook exposes a route which is added to Autoscan's main router.

If one wants to configure a HTTPTrigger with multiple distinct configurations,
then these configurations MUST provide a field called `Name` which uniquely identifies the trigger.
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
    - name: sonarr-docker   # /triggers/sonarr-docker
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
