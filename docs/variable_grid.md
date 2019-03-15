---
id: variable_grid
title: Variable Grid
sidebar_label: Variable Grid
---

InMAP is equipped with a variable-horizontal-resolution computational grid.
InMAP, like other air quality models, divides its spatial modeling domain into grid cells in three dimensions and performs computations on each grid cell, assuming everything is evenly mixed within each individual grid cell.

## Grid spatial extent

### Horizontal

The horizontal spatial extent of the grid is specified by the `VarGrid.VariableGridXo` and `VarGrid.VariableGridYo` variables, combined with the `VarGrid.VariableGridDx` and `VarGrid.VariableGridDy` configuration variables, and with the `VarGrid.Xnests` and `VarGrid.Ynests` configuration variables. The `VarGrid.VariableGridXo` and `VarGrid.VariableGridYo` variable specify the location of the south-west corner of the grid, and the `VarGrid.VariableGridDx` and `VarGrid.VariableGridDy` variables specify the edge lengths of the largest grid cells.
The `VarGrid.Xnests` and `VarGrid.Ynests` are both specified as lists of numbers, where
the first number in each list specifies the number of the largest grid cells in the X and Y directions

For example, suppose we set `VarGrid.VariableGridXo=-50000000.0`, `VarGrid.VariableGridDx=100000.0`, and `VarGrid.Xnests=[100,4,5,5]`.
This configuration specifies in the X-direction the grid starts at -50000000.0 m, is comprised of 100 cells, each one being 100,000 m in edge-length (assuming that the native units of the grid are meters).
Therefore, the right edge of the grid is specified to be at `-50000000.0 + 100000.0 * 100 = -40000000`.

### Vertical

The vertical spatial extent and resolution are specified by the vertical extent and resolution of the chemical transport model used to create InMAP inputs and are not directly configurable by the user.

## Grid resolution

In InMAP and similar models, there is a trade-off between spatial resolution and computational intensity: smaller grid cells provide more realism about spatial variability in pollution concentrations than do larger grid cells, but simulations with smaller grid cells also require more computational time and resources.
Computational time is proportional to the number of grid cells in a simulation.
It is also approximately inversely proportional to the edge length of the smallest grid cell, because the model equations are solved by taking steps forward in time, and the size of the step that can be taken is proportional to the size of the grid cell.
So, because the number of grid cells in a simulation is proportional to the square of the grid cell edge length, the computational time required for a simulation is proportional to the cube of the edge length.
In other words, decreasing the edge length of the grid cells by a factor of three would increase the computational intensity of the simulation by a factor of 27.

### Variable Grid Resolution Algorithms

Unlike a lot of models, InMAP's grid cells don't all need to be the same size, so it is able to provide high spatial resolution in areas where it is important to have high spatial resolution and lower resolution elsewhere.
Users of InMAP can choose between two algorithms for deciding the size of grid cells.
Brief descriptions of the two algorithms are below; detailed information is in an [Appendix](https://journals.plos.org/plosone/article/file?type=supplementary&id=info:doi/10.1371/journal.pone.0176131.s018) to a journal article describing InMAP.

### Static grid resolution algorithm

The static algorithm chooses the size of all grid cells before the simulation starts, and does not change the size of any grid cells while the simulation is running. Grid cell sizes are chosen based on the number of people in the grid cell, where any grid cell with more people than specified by the `VarGrid.PopThreshold` configuration variable or containing any region with population density greater than specified by the `VarGrid.PopDensityThreshold` configuration variable (in units of people per the square of the native length units of the grid [e.g., meters or degrees]) is split into smaller grid cells.
This algorithm is applied to all grid cells within the number of vertical layers from ground-level specified by the `VarGrid.HiResLayers` configuration variable.
All grid cells above that layer are kept at the lowest possible resolution.

To choose this algorithm, users can set the `static` configuration variable to `true`.
The static grid must either be created ahead of time using the `inmap grid` command or at the beginning of the simulation by setting the `creategrid` configuration variable to `true`.

### Dynamic grid resolution algorithm

The dynamic algorithm allows the size of the grid cells to change while the simulation is running.
At periodic intervals, the algorithm splits grid cells where the product of spatial gradients in population and simulated concentrations are above the threshold specified by the `VarGrid.PopConcThreshold` configuration variable into smaller grid cells.
This algorithm is applied to all grid cells (using the population on the ground below each cell), regardless of whether the grid cells are at ground level or elevated.

Because the units of the threshold are non-intuitive, in practice it often works best to use a default value or determine a suitable value by trial-and-error. The table below shows tradeoffs between the threshold value, calculated population exposure to PM2.5, and the time required to run the simulation. This table is reproduced from the supporting information of [a paper]() containing additional information regarding tradeoffs between grid cell size, simulation run-time, and estimated population-exposure to pollution. This table is valid for the default continental US spatial domain with an emissions scenario including all US emissions; it may not apply for other domains or emissions scenarios.

Size\* | # Cells | Run time (hrs) | `VarGrid.PopConcThreshold` | Exposure\*\*
---|---|---|---|---
69.1 | 1,818 | 0.6 | 5.00E-07 | 6.6
39.6 | 3,999 | 0.8 | 5.00E-08 | 7.0
21.1 | 10,599 | 1.3 | 5.00E-09 | 7.4
11.0 | 34,755 | 3.0 | 5.00E-10 | 7.9
5.9 | 103,098 | 10.7 | 5.00E-11 | 8.3

\*Population-weighted mean grid cell edge length

\*\*Population-weighted mean PM2.5 concentration (ug/m³).

InMAP uses the dynamic grid resolution by default, no specific action is required to select it.

### Allowed grid cell sizes

The allowed grid cell sizes are specified by the `VarGrid.VariableGridDx` and `VarGrid.VariableGridDy` configuration variables, combined with the `VarGrid.Xnests` and `VarGrid.Ynests` configuration variables. As described above, the `VarGrid.VariableGridDx` and `VarGrid.VariableGridDy` variables specify the edge lengths of the largest grid cells.
The `VarGrid.Xnests` and `VarGrid.Ynests` are both specified as lists of numbers.
The first number in each list specifies the number of the largest grid cells in the X and Y directions, and the remaining numbers in the list specify the number of smaller grid cells each larger grid cell can be split into.

For example, suppose we set `VarGrid.VariableGridDx=100000.0` and `VarGrid.Xnests=[100,4,5,5]`.
This configuration specifies in the X-direction the grid is comprised of 100 cells, each one being 100,000 m in edge-length (assuming that the native units of the grid are meters).
It also specifies that each of the 100km grid cells can be split into 4 25km grid cells, each of the 25km grid cells can then be split into 5 5km grid cells, and each of the 5 km grid cells can be split into 5 1 km grid cells.
So the allowed grid cell sizes are 100, 25, 5, and 1 km.

In general it is not recommended to use grid cell sizes < 1 km for models such as InMAP that use simple parameterizations for boundary-layer convective mixing.
