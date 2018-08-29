# Read.as

Read.as is a free and open-source long-form reader built on open web protocols (specifically ActivityPub). It helps you gather and curate things to read in a peaceful space.

It's written in Go (golang), and aims to use (minimal) plain Javascript on the frontend.

You can support the development of this project [on Patreon](https://www.patreon.com/thebaer) or [Liberapay](https://liberapay.com/writeas), or by becoming a [Write.as subscriber](https://write.as/subscribe).

## Features

* Read `Article`s from the fediverse
* Follow fediverse users via ActivityPub
* Single-user mode

## Requirements

* OpenSSL
* MySQL

**Additional requirements for development**

* [Go 1.10+](https://golang.org/dl/)
* [Node.js](https://nodejs.org/en/download/)

## Getting Started

```bash
# Set up database
# - Log into MySQL and run:
# CREATE DATABASE readas;
# - Import the schema:
mysql -u YOURUSERNAME -p readas < schema.sql

# Install the application
go get github.com/writeas/Read.as
cd $GOPATH/src/github.com/writeas/Read.as
make install

# Option 1: quick start
export RA_MYSQL_CONNECTION="YOURUSERNAME:YOURPASSWORD@tcp(localhost:3306)/readas"
readas -h "http://localhost:8080" -p 8080

# Option 2: configure and run
# Edit configuration
cp config.example.json config.json
vi config.json

# Create initial account
readas --user matt --pass hunter2

# Launch server
readas
```

### Configuration

`host` or the `-h` option should be the public-facing URL your site is hosted at, including the scheme, and without a trailing slash.

`port` or the `-p` option will be the port your server runs on. In production, add a reverse proxy like nginx in front of the app and point to `localhost:PORT`.

For `mysql_connection`, replace `YOURUSERNAME` and `YOURPASSWORD` with your MySQL authentication information, and `readas` with your database name.

By default, you'll see your site at `localhost:8080`. Be sure to update the `host`/`-h` option accordingly when running locally.

### Customizing

Go to the `users` table in your database to update your account's display name and summary.

## Deployment

**Use in production at your own risk.** This is very early software. Things will change and permanently break without notice, and support is minimal or non-existent while in version **0.x**.

Run:

```
./build.sh
```

Then copy the generated `build` directory to your server, into a place like `/var/app/readas`. Add a reverse proxy like Nginx, export your production `RA_MYSQL_CONNECTION` string on your machine, and from your install directory, run `readas -h "https://yourdomain.com"`.

## Development

After updating styles, run `make`.

After changing any code, run `go install ./cmd/readas && readas -h "http://localhost:8080"`

## Contributing

Feel free to open issues for any bugs you encounter, and submit any pull requests you think would be useful. To request features and discuss development, please see [our forum](https://discuss.write.as).
