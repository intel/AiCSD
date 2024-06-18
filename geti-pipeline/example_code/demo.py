########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

import cv2
from zipfile import ZipFile
from geti_sdk.deployment import Deployment
from geti_sdk.utils import show_image_with_annotation_scene
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import os
from json import dumps
import sys
import base64
import shutil

hostName = "0.0.0.0"
serverPort = 8080

class MyServer(BaseHTTPRequestHandler):
    
    # Normalize the path to remove any path traversal characters
    def safe_path(self,base_path,input):        
        safe_name = os.path.normpath(os.path.join(base_path, input))                
        
        if not safe_name.startswith(base_path):
            raise ValueError("Invalid directory path")

        return os.path.join(base_path, safe_name)

    def _send_cors_headers(self):
      self.send_header("Access-Control-Allow-Origin", "*")
      self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
      self.send_header("Access-Control-Allow-Headers", "x-api-key,Content-Type")

    def send_dict_response(self, d):
      self.wfile.write(bytes(dumps(d), "utf8"))

    def do_OPTIONS(self):
      self.send_response(200)
      self._send_cors_headers()
      self.end_headers()

    def do_GET(self):
        
        # List models available
        dir_list = os.listdir("../models")
        print(dir_list)
        self.send_response(200)
        self._send_cors_headers()
        self.send_header('Content-type', 'application/json')
        self.end_headers()        
        self.wfile.write(json.dumps({'models': dir_list}).encode('utf-8'))

    def do_POST(self):
        print("POST TRIGGERED")
        if self.path == '/' :
            try:
                content_length = int(self.headers['Content-Length']) 
                body = self.rfile.read(content_length)
                jsonObj = json.loads(body)                        
            except:
                self.send_response(400)
                self._send_cors_headers()
                print("Bad request")
                self.end_headers()
                return
            try:
                # Step 1: Load the deployment
                deployment = Deployment.from_folder("../../models/"+jsonObj["ModelName"]+"/deployment")

                # Step 2: Load the sample image
                image = cv2.imread(jsonObj["InputFileLocation"])
                image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)

                # Step 3: Send inference model(s) to CPU
                deployment.load_inference_models(device="CPU")

                # Step 4: Infer image
                prediction = deployment.infer(image_rgb)

                # Step 5: Visualization     
                show_image_with_annotation_scene(image_rgb, prediction, filepath=jsonObj["OutputFileFolder"], show_results=False )
            except Exception as e:
                self.send_response(500)
                self._send_cors_headers()
                print("An error ocurred while processing pipeline: {}".format(e))
                self.end_headers()
                return

            self.send_response(200)
            self._send_cors_headers()      
            self.send_header('Content-type', 'application/json')        
            self.end_headers()
            self.wfile.write(json.dumps({'name': prediction.annotations[0].labels[0].name, 'probability': prediction.annotations[0].labels[0].probability}).encode('utf-8'))
        elif self.path == "/upload":    
            try:
                content_length = int(self.headers['Content-Length'])
                body = self.rfile.read(content_length).decode('utf-8')
                jsonObj = json.loads(body)     
                name = jsonObj["Name"]
                modelType = jsonObj["Type"]
                dir_path = self.safe_path('/models', name)
                print(dir_path)
                if not os.path.exists(dir_path):
                    os.makedirs(dir_path)
                    
                    file_data = jsonObj["Zip"]
                    data_start_idx = file_data.index(',') + 1
                    base64_data = file_data[data_start_idx:]
                    text_file = open(os.path.join(dir_path, "temp.zip"), "wb")
                    text_file.write(base64.b64decode(base64_data))
                    text_file.close()
                                   
                    with ZipFile(os.path.join(dir_path, "temp.zip"), 'r') as zip_ref:
                        zip_ref.extractall(dir_path)
                         
                    items = os.listdir(dir_path)
                    for item in items:
                        item_path = self.safe_path(dir_path, item)
                    
                    if modelType == "geti":  
                        if os.path.isdir(item_path) and item != "deployment":
                         shutil.rmtree(item_path)
                        elif item != "deployment":
                         os.remove(item_path)
                    elif modelType == "ovms":
                        file = "/models/config.json"
                        with open(file, "r+") as f:
                          file_data = json.load(f)
                          new_data = {"config": {"name": name, "base_path": dir_path}}
                          file_data["model_config_list"].append(new_data)
                          f.seek(0)
                          json.dump(file_data, f, indent = 4)
                          print("Done updating config.json")

                    self.send_response(200)
                    self._send_cors_headers()
                    self.send_header('Content-type', 'application/json')    
                    self.end_headers()
                    self.wfile.write(json.dumps({'success': True}).encode('utf-8'))
                    print('done')
                else:
                    print(f"Model name {name} already exists")
                    self.send_response(500)
                    self.end_headers()
            except Exception:
                print(sys.exc_info()[2])
                self.send_response(500)                
                print("An error occurred while unzipping the file")
                self.end_headers()
                return
        else:
            self.send_response(400)
            print("Bad request 2")
            print("Invalid ENDPOINT: " + self.path)

if __name__ == "__main__":
    webServer = HTTPServer((hostName, serverPort), MyServer)
    print("Server started http://%s:%s" % (hostName, serverPort))
    webServer.serve_forever()