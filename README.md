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
| `-b, --body` | Request body: raw text/JSON, `@path` to read from a file, or `-` to read from stdin |
| `-d, --data` | Body field as `key=value`, repeatable; builds a JSON object |
| `--form` | Body field as `key=value`, repeatable; sends `application/x-www-form-urlencoded` |
| `-F, --multipart` | Multipart field as `key=value`, or `key=@path` to upload a file, repeatable; sends `multipart/form-data` and streams the file |
| `--query` | Query parameter as `key=value`, repeatable; merged into the URL and encoded |
| `-H, --header` | Request header as `Key: Value` or JSON, repeatable |
| `-o, --output` | Write the response body to this file instead of printing it |
| `-a, --auth` | Basic auth credentials as `user:pass` |
| `--timeout` | Request timeout (default `30s`) |
| `--retry` | Retry transient failures (network errors, 429, 5xx) up to this many times |
| `--retry-delay` | Base delay before a retry (default `500ms`); doubles each attempt, and `Retry-After` wins when present |
| `-k, --insecure` | Skip TLS certificate verification |
| `--proxy` | Send the request through this proxy, e.g. `http://127.0.0.1:8080` |
| `--no-redirect` | Do not follow redirects |
| `-f, --fail` | Exit with a non-zero status code on HTTP error responses (status >= 400) |
| `-i, --include` | Include the response status line and headers in the output, before the body |
| `-v, --verbose` | Print the outgoing request and response headers to stderr |
| `--raw` | Write the body verbatim: no JSON formatting, no binary-output guard |
| `--no-color` | Disable colored output (also auto-disabled when stdout isn't a terminal, and honors the `NO_COLOR` env var) |
| `--version` | Print the version and exit |
| `--help` | Print a full, up-to-date usage message and exit |

Only one of `--body`, `--data`, `--form`, or `--multipart` may be given —
each implies a different encoding.

### Output

stdout carries the response body and nothing else; the status line, save
notices, retry notices, and `-v` diagnostics all go to stderr. That means
the body pipes cleanly:

```bash
  ./http-client -m GET -u https://api.example.com/users | jq '.[0].name'
```

When stdout is a terminal, a JSON body is pretty-printed with sorted keys,
and a binary body is replaced by a one-line summary rather than scrambling
the terminal. When stdout is redirected or piped, the body is passed
through byte for byte, so parsers and files get exactly what the server
sent. `--raw` forces that verbatim output anywhere.

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

7- Download to a file (streamed, so the size doesn't matter):

```bash
  ./http-client -m GET --url http://example.com/big.zip -o big.zip
```

8- Fail a script on HTTP errors, with basic auth and a short timeout:

```bash
  ./http-client -m GET -u http://example.com/protected -a user:pass -f --timeout 5s
```

9- Add query parameters without escaping them by hand:

```bash
  ./http-client -m GET -u http://example.com/search --query "q=blue suede shoes" --query page=2
```

10- Post a url-encoded form:

```bash
  ./http-client -m POST -u http://example.com/login --form user=ada --form password=hunter2
```

11- Upload a file as multipart/form-data:

```bash
  ./http-client -m POST -u http://example.com/upload -F description="monthly export" -F data=@report.csv
```

12- Retry a flaky endpoint, showing the response headers:

```bash
  ./http-client -m GET -u http://example.com/flaky --retry 3 -i
```

13- Run the tool with `--help` to get a full, up-to-date usage message.

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
