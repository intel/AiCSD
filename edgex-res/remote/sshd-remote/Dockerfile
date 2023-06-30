#  ----------------------------------------------------------------------------------
#  Copyright 2023 Intel Corporation
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#
#  ----------------------------------------------------------------------------------

FROM debian:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends openssh-server && \
    rm -fr /var/lib/apt/lists/*

# Allow the openssh client to specify IP address from which connections to the port are allowed
RUN echo 'GatewayPorts clientspecified' >> /etc/ssh/sshd_config

# SSH login fix. Otherwise user is kicked off after login
# pam_loginuid is used to set the loginuid audit attribute of a process when a user login through SSH
RUN sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd

RUN mkdir /root/.ssh && chmod 700 /root/.ssh
COPY authorized_keys /root/.ssh/authorized_keys
RUN chmod 400 /root/.ssh/authorized_keys

CMD [ "sh" , "-c", "mkdir /var/run/sshd; exec /usr/sbin/sshd -D" ]
