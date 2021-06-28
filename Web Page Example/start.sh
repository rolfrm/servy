SERVY=../servy
HOST=http://localhost:8888
$SERVY $HOST/index.html cat index.html&
$SERVY $HOST/hash sha1sum&
$SERVY $HOST/code.js cat code.js
