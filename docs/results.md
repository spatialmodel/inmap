---
id: results
title: Output Results
sidebar_label: Output Results
---

InMAP calculates the changes in concentrations of PM2.5, its components, and its precursor pollutants caused by a scenario of changes in emissions specified by a user.

It also allows to the user to use the quantities that are natively calculated by the model (listed [here](output_options.md)) as building blocks to create other outputs using equations.
This behavior is specified with the `OutputVariable` configuration variable.

## OutputVariables

The `OutputVariables` configuration variable allows users to use equations to create their own outputs. Each equation consists of a new output variable name specified on the left, followed by an equals sign and an expression defining the new variable based on already-existing variables on the right.

For example, if we wanted to output a variable which was twice the average wind speed, we could specify:

```
[OutputVariables]
WindSpeed2 = "WindSpeed * 2"
```

It is important to note that variable names must be 11 characters long or less.

For a more useful output, we could specify population-weighted concentration:

```
[OutputVariables]
TotalPM25 = "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA"
PopWtd = "TotalPM25 * TotalPop / sum(TotalPop)"
```

Note that equations can include variables that are defined in another equation.
There are also some built-in equations, such as sum().
See the [API documentation](https://godoc.org/github.com/spatialmodel/inmap#NewOutputter) for a complete list of built-in functions.

We can also calculate differences in population-weighted concentrations among race-ethnicity groups:

```
[OutputVariables]
TotalPM25 = "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA"
whiteExp = "TotalPM25 * WhiteNoLat / sum(WhiteNoLat)"
blackExp = "TotalPM25 * Black / sum(Black)"
diff = sum(whiteExp - blackExp)
```

Or we can calculate health impacts from PM2.5 exposure, for example using a [Cox proportional hazards model](https://en.wikipedia.org/wiki/Proportional_hazards_model) and assuming a 6% increase in overall mortality for every 10 μg/m³ increase in PM2.5 concentration (based on the work of Krewski and colleagues, [2009](https://www.healtheffects.org/publication/extended-follow-and-spatial-analysis-american-cancer-society-study-linking-particulate)):

```
[OutputVariables]
TotalPM25 = "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA"
deaths = "(exp(log(1.06)/10 * TotalPM25) - 1) * TotalPop * allcause / 100000"
```

Although `OutputVariables` can be specified as an environment variable or command-line argument, it typically works best to specify them within a configuration file.
If a user wants to specify them as a environment variable or command-line argument, however, the entire set of variables must be converted to [JSON](https://www.json.org/) format. For example, the example above as a command-line argument would be:

    --OutputVariables="{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"deaths\":\"(exp(log(1.06)/10 * TotalPM25) - 1) * TotalPop * allcause / 100000\"}\n"
