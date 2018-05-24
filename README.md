# SMS Inbound Mock test tool, golang

This is a mock server for HORISEN AG premium transit SMS API: https://www.horisen.com/en/help/api-manuals/premium-transit


## Running the example

The example requires a working Go development environment. The [Getting
Started](http://golang.org/doc/install) page describes how to install the
development environment.

Once you have Go up and running, you can download, build and run the example
using the following commands.

```bash
go get github.com/gorilla/websocket
git clone git@github.com:rukavina/sms-inbound-mock-go.git
cd sms-inbound-mock-go
./install_dev.sh
```

Next time you need to start the server and test bot just run:

```bash
./start_dev.sh
```

In order to start the server only:

```bash
./sms-inbound-mock-go
```

To use the chat example, open http://localhost:9200/ in your browser.

## Configuration

The mock server is pre-configured to work with simple php sms bot in directory `test_client`. You can easily set in UI the URL of your SMS bot. `/public/js/config.js` holds some default options for the UI.

If you need to change the port or host where your mock server is running at, open `start_dev.sh` and replace the line 

```bash
./sms-inbound-mock-go
```

with 

```bash
./sms-inbound-mock-go --addr=mydomain:myport
```