# AWSBeats release tool

AWS Lambda is being invoked every N minutes and checks if there is a new Beats release. If so, it creates a new release.

## Install dependencies

```
npm install
```

## Deploy

```
sls deploy
```

## Run locally

```
GITHUB_TOKEN=boom sls invoke local -f cron
```
