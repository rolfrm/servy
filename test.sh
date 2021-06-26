log_success() {
  printf "${GREEN}✔ %s${NORMAL}\n" "$@" >&2
}

log_failure() {
  printf "${RED}✖ %s${NORMAL}\n" "$@" >&2
}
assert_eq() {
  local expected="$1"
  local actual="$2"
  local msg="${3-}"
  if [ "$expected" == "$actual" ]; then
    return 0
  else
    [ "${#msg}" -gt 0 ] && log_failure "$expected == $actual :: $msg" || true
    return 1
  fi
}

./servy http://localhost:54545/echo1 echo 1&
ID1=$!
curl -X GET http://localhost:54545/echo1

./servy http://localhost:54545/echo2 echo 2&
ID2=$!
./servy http://localhost:54545/zero dd if=/dev/zero bs=1000 count=1000&
ID3=$!
./servy http://localhost:54545/sha1sum sha1sum&
ID4=$!

until curl -X GET http://localhost:54545/echo2
do
  echo "Try again"
done


A=`curl -X GET http://localhost:54545/echo2`
B=`curl -X GET http://localhost:54545/echo1`

#time curl -X GET -Z -P 16 --silent  http://localhost:54545/echo1?iteration=[1-100000] -o /dev/null

#curl -X GET http://localhost:54545/zero -o /dev/null&
#Z1 = $!
#curl -X GET http://localhost:54545/zero -o /dev/null&
#Z2 = $!
#curl -X GET http://localhost:54545/zero -o /dev/null
D=`curl -X GET http://localhost:54545/zero --data-binary -v | curl -X POST -T /dev/stdin http://localhost:54545/sha1sum -v| cut -d " " -f 1`

#curl -X POST http://localhost:54545/sha1sum

C=`dd if=/dev/zero bs=1000 count=1000|sha1sum`
C2=`cat /dev/null|sha1sum| cut -d " " -f 1`
assert_eq $A 2 "CMP"
assert_eq $B 1 "CMP"
assert_eq $D $C "CMP"

kill $ID1
sleep 0.5
#curl -X GET http://localhost:54545/echo2

kill $ID2 $ID3 $ID4
