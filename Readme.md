The simplest way to create a web server. Transform executables into web endpoints.


```sh
# start a server and serve "Hello World" to index.html
./servy http://localhost:8888/index.html echo "<html><body><h1>Hello World</h1></body></html>"

# serve the date. Try accessing http://localhost:8888/date
./servy http://localhost:8888/date date

# lets try to add an argument. try accessing http://localhost:8888/date?TZ=America/New_York
./servy http://localhost:8888/date sh -c "date"
#TZ=x sets environment variables.

#http://localhost:8888/echo?ARG=hello_world
./servy http://localhost:8888/echo sh -c "echo \$ARG"


# For a more extreme use case do this:
./servy http://localhost:8888/sha1 sha1sum

#now from a different console do
curl -X POST -T ./main.go http://localhost:8888/sha1

```
## Build

```sh
go build
```

## SSL

Example of using SSL certificate to enable https endpoints.
```sh
openssl req -x509 -nodes  -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
export SERVY_CERT_FILE=./cert.pem 
export SERVY_KEY_FILE=./key.pem 
./servy https://localhost:8899/ls ls
```
