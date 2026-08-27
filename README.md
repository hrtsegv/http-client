# HTTP Client

[![CI](https://github.com/hrtsegv/http-client/actions/workflows/ci.yml/badge.svg)](https://github.com/hrtsegv/http-client/actions/workflows/ci.yml)

HTTP Client is a command-line tool for executing HTTP requests. It supports various HTTP methods such as GET, POST, PUT, DELETE, PATCH, OPTIONS, and HEAD.

## Screenshots:

![http-client](./http-client.png)

## Installation

1. Clone the repository:

   ```shell
   git clone https://github.com/hrtsegv/http-client.git
   ```
2. Build the project:
   ```shell
   cd http-client
   go build
   ```

## Usage

The general format of the command is:
  ```shell
  ./http-client -m [HTTP_Method] -u [URL] [flags]
  ```

### Flags

| Flag | Description |
| --- | --- |
| `-m, --http-method` | HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS (required) |
| `-u, --url` | Request URL (required). A missing scheme defaults to `https://` |
| `-b, --body` | Request body: raw text/JSON, `@path` to read from a file, or `-` to read from stdin. Mutually exclusive with `--data` |
| `-d, --data` | Body field as `key=value`, repeatable; builds a JSON object. Mutually exclusive with `--body` |
| `-H, --header` | Request header as `Key: Value` or JSON, repeatable |
| `-o` | Write the response body to this file |
| `-a, --auth` | Basic auth credentials as `user:pass` |
| `--timeout` | Request timeout (default `30s`) |
| `-k, --insecure` | Skip TLS certificate verification |
| `--no-redirect` | Do not follow redirects |
| `-f, --fail` | Exit with a non-zero status code on HTTP error responses (status >= 400) |
| `-v, --verbose` | Print the outgoing request and response headers to stderr |
| `--no-color` | Disable colored output (also auto-disabled when stdout isn't a terminal, and honors the `NO_COLOR` env var) |
| `--version` | Print the version and exit |
| `--help` | Print a full, up-to-date usage message and exit |

## Examples

1- Send an HTTP GET request:

```bash
  ./http-client -m GET --url http://example.com
```

2- Send an HTTP POST request with a JSON body:
```bash
  ./http-client -m POST -u http://example.com -b '{"key1":"value1","key2":"value2"}'
```

3- Send an HTTP POST request with `-d` fields instead of hand-writing JSON:
```bash
  ./http-client -m POST -u http://example.com -d key1=value1 -d key2=value2
```

4- Send an HTTP POST request with headers:
```bash
  ./http-client -m POST -u http://example.com -H "Content-Type: application/json" -H "Authorization: Bearer token" -b '{"key1":"value1"}'
```

5- Send a body from a file, or from stdin:
```bash
  ./http-client -m POST -u http://example.com -b @payload.json
  cat payload.json | ./http-client -m POST -u http://example.com -b -
```

6- Send an HTTP DELETE request:

```bash
  ./http-client -m DELETE -u http://example.com
```

7- Send an HTTP GET request and save results to an output file:

```bash
  ./http-client -m GET --url http://example.com -o data.json
```

8- Fail a script on HTTP errors, with basic auth and a short timeout:

```bash
  ./http-client -m GET -u http://example.com/protected -a user:pass -f --timeout 5s
```

9- Run the tool with `--help` to get a full, up-to-date usage message.

## Development

```bash
go build ./...          # build
go test ./...            # run tests
go vet ./...              # static checks
golangci-lint run ./...   # lint (see .golangci.yml)
```

## Contributions

Contributions are welcome! If you would like to contribute to this project, please follow these steps:

   1- Fork the repository.
   
   2- Create a new branch for your feature or bug fix.
   
   3- Make the necessary changes and commit them.
   
   4- Push your changes to your fork.
   
   5- Submit a pull request describing your changes.

## License

This project is licensed under the [MIT License](https://github.com/hrtsegv/http-client/blob/main/LICENSE). See the [LICENSE](https://github.com/hrtsegv/http-client/blob/main/LICENSE) file for details.
