The simplest way to create a web server. Transform executables into web endpoints.


```sh
# See the example.yaml file
./servy example.yaml
```

See the web page example for more information.
## Build

```sh
go build
```

## SSL

Example of using SSL certificate to enable https endpoints.
```sh
openssl req -x509 -nodes  -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
# These variables can also be set in your configuration file
export SERVY_CERT_FILE=./cert.pem 
export SERVY_KEY_FILE=./key.pem 
./servy configuration.yaml
```
