########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################
FROM grafana/grafana:8.5.3 AS builder

# Disable Login form or not
ENV GF_AUTH_DISABLE_LOGIN_FORM "false"
# Allow anonymous authentication or not
ENV GF_AUTH_ANONYMOUS_ENABLED "true"
# Role of anonymous user
ENV GF_AUTH_ANONYMOUS_ORG_ROLE "${GRAFANA_USER}"

USER root
RUN apk add --no-cache dumb-init=1.2.5-r1

USER "${GRAFANA_USER}"

# Add provisioning
COPY provisioning/ /etc/grafana/provisioning
# Add configuration file
COPY grafana.ini /etc/grafana/grafana.ini

# Note that exec form is required for effective use of dumb-init:
# https://github.com/Yelp/dumb-init#usage
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["sh", "-c", "exec /run.sh"]