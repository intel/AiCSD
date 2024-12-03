########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

FROM python:3.9-bullseye
RUN apt-get update && apt-get install --no-install-recommends ffmpeg=7:4.3.8-0+deb11u1 libsm6=2:1.2.3-1 libxext6=2:1.3.3-1.1 -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
COPY . .
WORKDIR /example_code
RUN python -m pip install --no-cache-dir -r requirements.txt
ENV PYTHONUNBUFFERED=1
EXPOSE 8080
CMD ["python", "demo.py"]
