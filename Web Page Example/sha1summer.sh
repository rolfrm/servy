#/bin/bash
while true
do
    len=$(./readint)
    if [ $? != 0 ]
    then
       exit 0
    fi
    dd if=/dev/stdin count=$len bs=1 of=/dev/stdout | sha1sum
    
done
