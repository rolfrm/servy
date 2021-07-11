#/bin/bash
./servy ws://localhost:8889/echo1 cat /dev/stdin&
./servy http://localhost:8889/index.html cat ./websocket.html
