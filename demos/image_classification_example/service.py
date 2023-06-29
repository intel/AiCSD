########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

import sys
sys.path.append("image_classification/python")

import numpy as np
import bentoml
from bentoml.io import NumpyNdarray
from bentoml.io import Text
import subprocess
import json
import image_classification
import helper
import logging

svc = bentoml.Service("image_classification", runners=[])

@svc.api(input=Text(), output=Text())
def classify(text: str) -> str:

    try:
     data = json.loads(text)
     print("Decoded json message received by the bentoml service: ",data)
     
     # Call image classification
     result = image_classification.classify(grpc_address=data["GatewayIP"], grpc_port=9001, input_name="0", output_name="1463", images_list="image_classification/input_images.txt")
     print("json result returned from pipeline: ",result)

     resultList = result
     
     if resultList["Status"] == "PipelineComplete":
      helper.send_pipeline_inference_results(data["JobUpdateUrl"],resultList, data["GatewayIP"])
      helper.send_pipeline_status(data["PipelineStatusUrl"], resultList["Status"], data["GatewayIP"])
      return resultList["Status"]
     else:
       raise Exception("Pipeline completed, but failed")   
    except Exception as e:
      try:
       print("Error occurred while handling the service: "+ str(e))
       infereneceResults = {"Status": "PipelineFailed"}
       helper.send_pipeline_inference_results(data["JobUpdateUrl"], infereneceResults, data["GatewayIP"])
       helper.send_pipeline_status(data["PipelineStatusUrl"], "PipelineFailed", data["GatewayIP"])
      finally:
       print(str(e))
       return "PipelineFailed"      
    

