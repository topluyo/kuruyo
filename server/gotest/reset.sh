#!/bin/bash
rm go.mod > /dev/null 2>&1
rm go.sum > /dev/null 2>&1
/usr/local/go/bin/go mod init app > /dev/null 2>&1
/usr/local/go/bin/go mod tidy > /dev/null 2>&1