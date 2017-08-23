# go-server-generator
Generate Go server code from swagger spec

## Usage instructions

You can use this generator in two different ways. If you don't care about breaking changes, just run the main file. Otherwise, it's better to vendor the package and generate a command specific for your project.

### When you don't care about breaking changes

**Install**:

```sh
go install github.com/fujitsueos/go-server-generator
```

**Generate the code**:

```sh
go-server-generator <path to your swagger file>
```

## When you care about breaking changes

**Get the repo**:

```sh
go get github.com/fujitsueos/go-server-generator
```

**Create a generator specific for your repo**:

```sh
go run $GOPATH/src/github.com/fujitsueos/go-server-generator/cmd/generate/main.go <path to your swagger file>
godep save <your repo relative to $GOPATH/src>/cmd/generate
```

Check in the generated command and the vendored dependency

**Generate the code**:

```sh
go run <your repo>/cmd/generate/main.go
```
