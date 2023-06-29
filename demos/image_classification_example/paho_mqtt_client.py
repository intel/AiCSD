########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

import paho.mqtt.subscribe as subscribe
import requests
import pprint
import yaml
import argparse
import os
import json
import logging

pp = pprint.PrettyPrinter(indent=4)

#read_yaml enables reading from config file & return results as a dictionary
def read_yaml(file_path):
    with open(file_path, "r") as f:
     return yaml.safe_load(f)

#on_message_print is the callback function to handle mqtt messages
def on_message_print(client, userdata, message):
    pp.pprint("Message topic:%s, Payload: %s" % (message.topic, message.payload))
    decoded_data= message.payload.decode("utf-8")

    gateway_ip = {"GatewayIP":config_values["gateway_ip"]}
    jsonMsg = json.loads(decoded_data)
    jsonMsg.update(gateway_ip)
    jsonMsgStr = json.dumps(jsonMsg)
    post_url = config_values["service"]["POST_url"] + ":" + config_values["service"]["port"] + "/" + config_values["service"]["service_func"] 
    pp.pprint("Sending POST request to: ")
    pp.pprint(post_url )
    requests.post(post_url, headers={"content-type":"application/json"}, data=jsonMsgStr).text


parser = argparse.ArgumentParser()
parser.add_argument("file_path")
args = parser.parse_args()
check_file = os.path.isfile(args.file_path)

if not check_file:
 pp.pprint("Config file doesnot exist")
 raise SystemExit(1)

config_values = read_yaml(args.file_path)
pp.pprint("configurations are: ") 
pp.pprint(config_values) 

#Subscribe to MQTT topic for inference results
subscribe.callback(on_message_print, config_values["mqtt"]["topic"], hostname=config_values["mqtt"]["hostname"], port=config_values["mqtt"]["port"])
