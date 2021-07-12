#/bin/bash
./servy ws://localhost:8889/echo1 cat /dev/stdin&
./servy ws://localhost:8889/sha1 bash -c ./sha1summer.sh&
./servy ws://localhost:8889/flen bash -c ./flen.sh&
./servy http://localhost:8889/index.html cat ./websocket.html
