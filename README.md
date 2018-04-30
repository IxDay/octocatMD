OctocatMD
=========

Locally render markdown files as Github does.
This is inspired by the excellent [grip](https://github.com/joeyespo/grip).
It provides a single binary which consists of an http server rendering markdown
on the fly.

Getting started
---------------

By default, `octocatmd` render the current directory at `localhost:5678`.
There's a few options displayed with `octocatmd -h`, like port, host or debug output.

How to
------

This project is a simple `golang` project, here is a simple list of command
to set things up.

```bash
export GOPATH="/tmp/octocatmd"
go get "github.com/ixday/octocatmd"
cd "${GOPATH}/src/github.com/ixday/octocatmd

# this will build a new binary in "${GOPATH}/bin"
make

# start a development server in debug mode on port 6789
$(make cmd) -d -p 6789
```
