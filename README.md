# Golang integration that transfers files from FTP to Nextcloud

## Context

My scanner (Brother ADS-1700W) can store documents on FTP servers but there is
no support for Nextcloud nor any generic WebDAV server.

This integration fixes this.

## Compilation

```sh
go build -o ftp-webdav-integration
```

## Pre-requisites

* Nextcloud instance
* FTP server

## Usage

```sh
podman-compose up -d
./ftp-webdav-integration config.yaml
```
