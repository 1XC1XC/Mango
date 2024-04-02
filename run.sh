#!/bin/bash

rm -rdf ./Mango
go build .
chmod +x ./Mango
./Mango "$@"
