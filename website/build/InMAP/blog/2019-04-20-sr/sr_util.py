# Ensure compatibility between python 2 and python 3
from __future__ import (absolute_import, division,
                        print_function, unicode_literals)
from builtins import *

import requests
import platform
import os
import stat
import tempfile
import json
import time
import subprocess
import geopandas as gpd
import shutil

def _download(url, file_name):
    # open in binary mode
    with open(file_name, "wb") as file:
        # get request
        response = requests.get(url)
        if not response.ok:
            raise Exception("Downloading file from {} failed".format(url))
        # write to file
        file.write(response.content)

_inmap_exe = None
_tmpdir = tempfile.TemporaryDirectory()

def run_sr(emis, model, output_variables, emis_units="tons/year"):
    """
    Run the provided emissions through the specified SR matrix, calculating the
    specified output properties.

    Args:
        emis: The emissions to be calculated, Needs to be a geopandas dataframe.

        model: The SR matrix to use. Allowed values:
            isrm: The InMAP SR matrix
            apsca_q0: The APSCA SR matrix, annual average
            apsca_q1: The APSCA SR matrix, Jan-Mar season
            apsca_q2: The APSCA SR matrix, Apr-Jun season
            apsca_q3: The APSCA SR matrix, Jul-Sep season
            apsca_q4: The APSCA SR matrix, Oct-Dec season

        output_variables: Output variables to be calculated. See
            https://inmap.run/docs/results/ for more information.

        emis_units: The units that the emissions are in. Allowed values:
            'tons/year', 'kg/year', 'ug/s', and 'μg/s'.
    """


    global _tmpdir
    global _inmap_exe

    model_paths = {
        "isrm": "/data/isrmv121/isrm_v1.2.1.ncf",
        "apsca_q0": "/data/apsca/apsca_sr_Q0_v1.2.1.ncf",
        "apsca_q1": "/data/apsca/apsca_sr_Q1_v1.2.1.ncf",
        "apsca_q2": "/data/apsca/apsca_sr_Q2_v1.2.1.ncf",
        "apsca_q3": "/data/apsca/apsca_sr_Q3_v1.2.1.ncf",
        "apsca_q4": "/data/apsca/apsca_sr_Q4_v1.2.1.ncf",
    }
    if model not in model_paths.keys():
        models = ', '.join("{!s}".format(k) for (k) in model_paths.keys())
        msg = 'model must be one of \{{!s}\}, but is `{!s}`'.format(models, model)
        raise ValueError(msg)
    model_path = model_paths[model]

    start = time.time()
    job_name = "run_aqm_%s"%start
    emis_file = os.path.join(_tmpdir.name, "%s.shp"%(job_name))
    emis.to_file(emis_file)

    arch = platform.machine()

    if arch == "x86_64": arch = "amd64"
    
    version = "1.9.0"
    
    if _inmap_exe == None:
        ost = platform.system()
        print("Downloading InMAP executable for %s               "%ost, end='\r')
        if ost == "Windows":
            _inmap_exe = os.path.join(_tmpdir.name, "inmap_%s.exe"%version)
            _download("https://github.com/spatialmodel/inmap/releases/download/v%s/inmap-v%s-windows-%s.exe"%(version, version, arch), _inmap_exe)
        elif ost == "Darwin":
            _inmap_exe = os.path.join(_tmpdir.name, "inmap_%s"%version)
            _download("https://github.com/spatialmodel/inmap/releases/download/v%s/inmap-v%s-darwin-%s"%(version, version, arch), _inmap_exe)
        elif ost == "Linux":
            _inmap_exe = os.path.join(_tmpdir.name, "inmap_%s"%version)
            _download("https://github.com/spatialmodel/inmap/releases/download/v%s/inmap-v%s-linux-%s"%(version, version, arch), _inmap_exe)
        else:
            raise(OSError("invalid operating system %s"%(ost)))
        os.chmod(_inmap_exe, stat.S_IXUSR|stat.S_IRUSR|stat.S_IWUSR)
    
    subprocess.check_output([_inmap_exe, "cloud", "start",
        "--cmds=srpredict",
        "--version=v%s"%version,
        "--job_name=%s"%job_name,
        "--memory_gb=3",
        "--EmissionUnits=%s"%emis_units,
        "--EmissionsShapefiles=%s"%emis_file,
        "--OutputVariables=%s"%json.dumps(output_variables),
        "--SR.OutputFile=%s"%model_path])

    # You can also run the simulation locally by replacing the above command with the below command:
    # subprocess.check_output([_inmap_exe, "srpredict",
    #    "--EmissionUnits=%s"%emis_units,
    #    "--EmissionsShapefiles=%s"%emis_file,
    #    "--OutputVariables=%s"%json.dumps(output_variables),
    #    "--SR.OutputFile=/path/to/isrm/file/you/downloaded.ncf"])
    # You can download the ISRM source receptor matrix itself from here: https://zenodo.org/record/3590127#.Yl2SSvPMKWA
    
    while True:
        try:
            status = subprocess.check_output([_inmap_exe, "cloud", "status", "--job_name=%s"%job_name]).decode("utf-8").strip()
            print("simulation %s (%.0f seconds)               "%(status, time.time()-start), end='\r')
            if status == "Complete":
                break
            elif status != "Running":
                raise ValueError(status)
        except subprocess.CalledProcessError as err:
            # ignore transient errors when checking status
            print(err.output)
        time.sleep(5)

    subprocess.check_call([_inmap_exe, "cloud", "output", "--job_name=%s"%job_name])
    output = gpd.read_file("%s/OutputFile.shp"%job_name)

    shutil.rmtree(job_name)
    subprocess.check_call([_inmap_exe, "cloud", "delete", "--job_name=%s"%job_name])

    print("Finished (%.0f seconds)               "%(time.time()-start))

    return output
