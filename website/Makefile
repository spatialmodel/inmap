#!/usr/bin/env bash

docusaurus :
	- docker run -it -v `pwd`/tempdir:/src --user $(id -u):$(id -g) mkenney/npm:debian npx docusaurus-init
	- sudo chown -R $$USER:$$USER tempdir
	- mv tempdir/website/node_modules .

update :
	- cd ../ && docker run -it -v `pwd`:/src mkenney/npm:debian /bin/bash -c "cd website && yarn run build"
	- sudo chown -R $$USER:$$USER build

.PHONY : clean
clean :
	- rm -rf node_modules tempdir
