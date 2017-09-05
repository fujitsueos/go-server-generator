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

## Special cases and limitations

When generating code out of a swagger spec, it is good to know what works and what doesn't. This is a short list of notable gotchas.

### Limitations

Big parts of the spec are not implemented because we can survive without them. Some notable examples:

- `schemes`, `consumes`, `produces`, `parameters`, `responses`, `securityDefinitions`, `security`, `tags` on top level are completely ignored by the generator, without warning.
- All type definitions *must* be in `definitions`. This implies that an endpoint cannot return an array of some type; we need to create a type alias in `definitions` for that array instead. (Main reason is that this ensures every type has a unique name, which makes code generation much easier.)
- Only a subset of validation rules is implemented. Using a validation rule that is not supported results in an error.
- Errors cannot use validation rules at all. (Errors are output only, so validation rules provide less value there.)
- It is not allowed to reference an object that has read-only properties from another type definition, except for arrays that serve as type-aliases only. (We generate two Go types for an object with read-only properties, one with the read-only properties and one with the rest. Doing this for the transitive closure of the type hierarchy referencing an object with read-only properties is cumbersome and doesn't provide much value.)

### Special cases

- When defining an error type, add `x-error: true` to the type definition. This makes sure that the type implements the Go Error interface.
- Every route can return 500 - Internal Server Error and every route that has input validation can return 400 - Bad Request. When you do not add the result type for these error for any route to the spec, it is assumed that their type is string. If you specify the type for at least one route, you need to specify the type for every route. The generator creates callbacks for each of the types that can be returned for these status codes (for all endpoints combined) that need to be implemented. If you make sure that every endpoint uses the same error type for 400 and the same for 500 (which is recommended), you only need to implement two methods.
