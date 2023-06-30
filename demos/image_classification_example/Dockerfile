########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

FROM python:3-slim
RUN pip install paho-mqtt==1.6.1 
RUN pip install requests==2.31.0
RUN pip install PyYAML==6.0
ENV PYTHONUNBUFFERED="true" PYTHONDONTWRITEBYTECODE=1

COPY paho_mqtt_client.py pythonClientConfig.yaml /

CMD [ "python", "./paho_mqtt_client.py", "pythonClientConfig.yaml" ]
