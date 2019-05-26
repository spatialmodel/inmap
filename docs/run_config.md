---
id: run_config
title: Running and Configuration
sidebar_label: Running and Configuration
---

## Running InMAP

InMAP can be run by opening a command line terminal and running the command `inmap` followed by any desired subcommands and command-line arguments, as in:

    inmap subcommand1 subcommand2 --arg1=x --arg2=y

If InMAP was downloaded from the [releases page](https://github.com/spatialmodel/inmap/releases) as a pre-compiled binary rather than compiled from source, you will need to navigate to the directory that InMAP was downloaded to and run the downloaded file, for example as in:

    ./inmapX.X.Xlinux-amd64 subcommand1 subcommand2 --arg1=x --arg2=y

for linux, or

    ./inmap1.4.2darwin-amd64 subcommand1 subcommand2 --arg1=x --arg2=y

for macOS, or

    inmapX.X.Xwindows-amd64.exe subcommand1 subcommand2 --arg1=x --arg2=y

for Windows.

The exact command may vary depending on your system configuration, and you may need to make the downloaded InMAP file executable before running it.

### Subcommands

The available InMAP subcommands are listed in the [InMAP command documentation](cmd/inmap.md), which also includes descriptions of what each command does and the available configuration settings.

## Configuration

InMAP has a number of settings that can be specified by a user, which we will call configuration variables.
InMAP configuration can be specified by a configuration file, environment variables, or command-line argument.
Configuration variables set by command-line arguments take first precedence, followed by configuration variables set by environment variables and finally by variables set using a configuration file. Any configuration variables that are not explicitly set take default values.

### Available settings and default values

Available configuration variables---and their default values---for each subcommand are described in the [command documentation](cmd/inmap.md).

### Configuration file

Configuration files are text files in the [TOML](https://github.com/toml-lang/toml) format. A configuration file can have any name and any file extension, although common choices are `inmap.toml` or `config.txt`. TOML configuration files can be edited using any text editor (e.g., including Microsoft Notepad), but on some systems the file extension may need to be changed to `.txt` for the system to recognize that the configuration file is in fact a text file.

In general, configuration variables can be specified in the the format `VarName = "Value"` for text variables, `VarName = 42` for integers, and `VarName = 42.0` for real-valued numbers.

For configuration values with a `.` in them, for example `VarGrid.VariableGridXo` and `VarGrid.VariableGridYo`, the part of the name before the `.` represents a category and the part of the name after the `.` represents a variable within that category. In the TOML configuration file, this must be specified in the following way:

``` TOML
[VarGrid]
VariableGridXo = 42.0
VariableGridYo = -42.0
```

For additional information, refer to the [TOML specification](https://github.com/toml-lang/toml), or to the [example configuration files](https://github.com/spatialmodel/inmap/tree/master/cmd/inmap).

### Environment variables

Configuration variables can also be set using environment variables, which override any variables set using a configuration file. Environment variables must have an `INMAP_` prefix followed by the variable name, for example `INMAP_VarGrid.VariableGridXo=42.0`. Users that do not have a strong understanding of what an environment variable is can safely skip this section.

### Command-line arguments

Finally, configuration variables can be set using command-line arguments, which override variables set in any other way. This can be done using the format `inmap subcommand1 subcommand2 --VarGrid.VariableGridXo=42.0`
