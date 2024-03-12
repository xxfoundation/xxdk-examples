# golang

An xxdk example written for Go.

## Running This Example

You must have a modern `go` installation:

https://go.dev/dl/

Our go version is:

```
% go version
go version
go version go1.21.5 linux/amd64
```

Start the example:

```
go run main.go
```

You will see dependencies installed then output written to the console.

## How This Example Was Built

We used `cobra-cli` to install a command line system:

```
go install github.com/spf13/cobra-cli@latest
cobra-cli init .

```

Then created a `go.mod` file with the following contents at the top:

```
module git.xx.network/xx_network/xxdk-examples/golang

go 1.21

toolchain go1.21.5
```

Then we added "client" xxdk library and cobra:

```
go get gitlab.com/elixxir/client/v4
go get github.com/spf13/cobra
```

And we made our modifications to `cmd/root.go`, following the example
from the client repository.
