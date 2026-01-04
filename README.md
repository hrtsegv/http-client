# HTTP Client

HTTP Client is a command-line tool for executing HTTP requests. It supports various HTTP methods such as GET, POST, PUT, DELETE, PATCH, OPTIONS, and HEAD.

## Screenshots:

![http-client](./http-client.png)

## Installation

1. Clone the repository:

   ```shell
   git clone https://github.com/knbr13/http-client.git
   ```
2. Build the project:
   ```shell
   cd http-client
   go build
   ```
## Usage

The general format of the command is:
  ```shell
  ./http-client -m [HTTP_Method] -b [body] -H [headers] -u [URL] -o [OUTPUT]
  ```
## Examples

1- Send an HTTP GET request:

```bash
  ./http-client -m GET --url http://example.com
```

2- Send an HTTP POST request with JSON body:
```bash
  ./http-client -m POST -u http://example.com -b '{"key1":"value1","key2":"value2"}'
```

3- Send an HTTP POST request with headers:
```bash
  ./http-client -m POST -u http://example.com -H "Content-Type: application/json" -H "Authorization: Bearer token" -b '{"key1":"value1"}'
```

4- Send an HTTP DELETE request:

```bash
  ./http-client -m DELETE -u http://example.com
```

5- Send an HTTP GET request and save results to output file:

```bash
  ./http-client -m GET --url http://example.com -o data.json
```

6- Run the tool with `--help` arg to get a helpful message explaining how to use this tool.


## Contributions

Contributions are welcome! If you would like to contribute to this project, please follow these steps:

   1- Fork the repository.
   
   2- Create a new branch for your feature or bug fix.
   
   3- Make the necessary changes and commit them.
   
   4- Push your changes to your fork.
   
   5- Submit a pull request describing your changes.

## License

This project is licensed under the [MIT License](https://github.com/knbr13/http-client/blob/main/LICENSE). See the [LICENSE](https://github.com/knbr13/http-client/blob/main/LICENSE) file for details.

