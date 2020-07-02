---
id: install
title: InMAP Installation
sidebar_label: Installation
---

## Getting InMAP

Go to [releases](https://github.com/spatialmodel/inmap/releases) to download the most recent release for your type of computer. For Mac systems, download the file with "darwin" in the name. You will need both the executable program and the input data ("evaldata_vX.X.X.zip"). All of the versions of the program are labeled "amd64" to denote that they are for 64-bit processors (i.e., all relatively recent notebook and desktop computers). It doesn't matter whether your computer processor is made by AMD or another brand, it should work either way.

### Compiling from source

You can also compile InMAP from its source code. The instructions here are specific to Linux or Mac computers; other systems should work with minor changes to the commands below. Refer [here](http://golang.org/doc/install#requirements) for a list of theoretically supported systems.

1. Install the [Go compiler](http://golang.org/doc/install), version 1.11 or higher. Make sure you install the correct version (64 bit) for your system. It may be useful to go through one of the tutorials to make sure the compiler is correctly installed.

3. Install the [git](http://git-scm.com/) version control program and the [GCC](https://gcc.gnu.org/) compiler if they are not already installed. If you are using a shared system or cluster, you may just need to load them with the commands `module load git` and `module load hg`. If you are using Windows or a Mac, it may work best to install them using [Anaconda](https://anaconda.org/).

4. Download and install the main program:

	``` bash
	git clone https://github.com/spatialmodel/inmap.git # Download the code.
	cd inmap # Move into the InMAP directory
	go build ./cmd/inmap # Compile the InMAP executable.
	```

	There should now be a file named `inmap` or `inmap.exe` in the current dirctory. This is the inmap executable file. It can be copied or moved to any directory of your choosing and run as described below in "Running InMAP".

5. Optional: run the tests:

	``` bash
	cd /path/to/inmap # Move to the directory where InMAP is downloaded,
	# if you are not already there.
	go test ./... -short
	```
