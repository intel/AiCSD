# Kibana

## Overview
Kibana is an open-source data visualization and exploration tool for reviewing logs and events. AiCSD uses Kibana to analyze and visualize container and service logs in an easy-to-use UI.

## Visit Kibana UI
Navigate to [http://localhost:5601](http://localhost:5601)

## Kibana Authentication
Authentication is currently enabled for Kibana. To log in, the username is `elastic` and the password is `ELASTIC_PASSWORD`. These authentication values were set in the `.env` file.

The Kibana UI creates two users, `kibana_system` and `beat_system`, as part of the bootstrapping of the ELK stack setup. These are not used for login purposes.
These are service accounts that allow the services to communicate properly and visualize the container and service logs.

## Resources

- [Kibana](https://www.elastic.co/kibana/).

BSD-3 License: See [License](../LICENSE.md).