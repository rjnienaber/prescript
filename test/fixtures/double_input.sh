#!/usr/bin/env bash

echo Hello!

echo -n "First number: "
read FIRST_NUM

sleep 2

echo -n "Second number: "
read SECOND_NUM

SUM=`expr $FIRST_NUM + $SECOND_NUM`
echo "Sum: $SUM"




