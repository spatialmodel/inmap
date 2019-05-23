#!/usr/bin/env bash

docusaurus :
	- docker run --rm -it -v `pwd`/tempdir:/src --user $(id -u):$(id -g) mkenney/npm:debian npx docusaurus-init
	- sudo chown -R $$USER:$$USER tempdir
	- mv tempdir/website/node_modules .

update :
	- cd ../ && docker run --rm -it -v `pwd`:/src mkenney/npm:debian /bin/bash -c "cd website && yarn run build"
	- sudo chown -R $$USER:$$USER build

host :
	- cd ../ && docker run --rm -p 3000:3000 -it -v `pwd`:/src mkenney/npm:debian yarn start

.PHONY : clean
clean :
	- rm -rf node_modules tempdir
