## PiPeek

The simple, lightweight Raspberry Pi monitoring tool.

![logo](https://github.com/alvaropp/PiPeek/blob/main/media/logo.png)


### Why PiPeek?

I wanted a simple, lightweight resource monitoring tool for my Raspberry Pis. The current standard tools are Grafana + Prometheus, but they are too heavy for my use case and take way too many resources on older Pis. PiPeek runs on Docker and takes up very little resources and disk space. That said, it's much less powerful and customizable than a proper monitoring stack, but that's OK for my use case.

Resource utilisation data is stored in a MongoDB Atlas database, which is free and very easy to set up. The dashboard is a simple client-side web app that displays the resource utilisation data in a few simple graphs, one for each monitored device.

<!-- ![]() -->


### Installation

PiPeek has two components: a logger and a dashboard visualisation, both running in Docker containers. The idea is that you run a logger on each device you want to monitor, and the dashboard on a single device which will serve the dashboard website.

#### Mongo database and API
1. Rename `env.example` to `.env`.
2. Set up a [MongoDB Atlas database](https://www.mongodb.com/atlas/database). Create a cluster, collection and database, and populate the following fields in the `.env` file: `MONGO_URI`, `MONGO_CLUSTER_NAME`, `MONGO_DATABASE`.
3. Activate Mongo's Data API for your database. This provides API access to your database, which is used to query the resource utilisation data from the dashboard. Note your app ID and API key, and populate the following fields in the `.env` file: `MONGO_APP_ID`, `MONGO_MONITOR_KEY`.

#### Configure logger
Give the device a name by setting the `MONGO_LOGGER_COLLECTION` variable in `.env`. This name must be unique for each device.
Set up the logging configuration by setting `NUM_SAMPLES`, `SAMPLE_INTERVAL_SECS`, `SLEEP_DURATION_SECS` in `.env`. These parameters control the number of samples to average at each logging round, the number of seconds between each sample, and the number of seconds to sleep between logging rounds, respectively. Default values are provided in `env.example`.

Build and run a container for the logger with `docker compose up -d pipeek-logger`. Do this on each device you want to monitor.

#### Configure monitor
As mentioned above, you only need to do this in a single device, which will serve the dashboard for you to look at. The monitor needs two environment variables: `MONGO_MONITOR_KEY` and `MONGO_MONITOR_COLLECTIONS`. `MONGO_MONITOR_KEY` is the key to your database's Data API and can be obtained in Mongo's website. `MONGO_MONITOR_COLLECTIONS` is a comma-separated list of the names of the devices you want to monitor. These must match the `MONGO_LOGGER_COLLECTION` names you set for each device you are monitoring (see previous section). Populate these fields in the `.env` file.

Build and run a container for the logger with `docker compose up -d pipeek-logger`. The dashboard will then be available at `http://IP:9999`.
