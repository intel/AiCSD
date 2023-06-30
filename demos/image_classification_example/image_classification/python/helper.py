########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

from typing import Dict, Any

import json
import requests
import logging

"""
inferenceResults should be infollowing json format
{
  "Status": "PipelineComplete",
  "QCFlags": "0",
  "OutputFileHost": "gateway",
  "OutputFiles": [
    {
      "Hostname": "gateway",
      "DirName": "/tmp/files/output",
      "Name": "test1out.tiff",
      "Extension": "tiff",
      "Status": "FileIncomplete"
    }
  ],
  "Results": "CellCount,25"
}
"""
def create_inference_response(results: Dict[str, Any], pipelineStatus: str) -> Dict[str, str]:
 jsonMsg = {
   "Status": "PipelineComplete",
   "QCFlags": "None", 
   "OutputFileHost": "gateway",
   "Results": results
  }
 
 return jsonMsg
 
def send_pipeline_status(pipelineStatusUrl: str, pipelineStatus: str, gatewayIP: str):
    try:
     requests.post(pipelineStatusUrl, pipelineStatus)
     print("Pipeline Status %s sent via POST request to %s " % (pipelineStatus, pipelineStatusUrl))     
    except Exception as e:
     print("Error occurred while sending pipeline status: "+ str(e))


def send_pipeline_inference_results(jobUpdateUrl: str, inferenceResults: str, gatewayIP: str):
    try:
     msg = inferenceResults
     x = requests.put(jobUpdateUrl, json=msg)
     print("Job updated via PUT request to ", jobUpdateUrl)
    except Exception as e:
     print("Error occurred while sending inference results: "+ str(e))
