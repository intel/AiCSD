# Filebeat

## Overview
Filebeat is an open-source analytics and interactive visualization web application. AiCSD leverages Kibana UI to display container and service logs captured by Filebeat and forwarded to Elasticsearch.

For more information on Filebeat, please refer to the official documentation [here](https://www.elastic.co/beats/filebeat).

## Filebeat Authentication
Authentication is currently enabled for Filebeat. The Elasticsearch UI supports a `beats_system` user created by the bootstrapping of the ELK stack setup. The `beats_system` user is one of which will be used by Filebeat to forward logs to Elasticsearch.

## Filebeat Over Logstash
AiCSD implements a variation of the popular log analytics stack containing Elasticsearch, Logstash, and Kibana (ELK). It replaces Logstash with Filebeat, eliminating Logstash's JVM installation requirement. In addition, Filebeat uses fewer resources on the installation machine. 

## Resources

- [General Information: Filebeat](https://www.elastic.co/beats/filebeat)
- [Filebeat Docs](https://www.elastic.co/guide/en/beats/filebeat/current/index.html)

INTEL CONFIDENTIAL: See [License](../LICENSE.md)
