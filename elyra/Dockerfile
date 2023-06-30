########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

ARG ELYRA_IMAGE=elyra/elyra:3.14.2
ARG BASE=${ELYRA_IMAGE}
FROM ${BASE} AS builder

USER root

RUN python -m pip install ovmsclient

ENTRYPOINT [ "start-elyra.sh"]
