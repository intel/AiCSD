# Elasticsearch

## Overview
A free, distributed search and analytics engine, Elasticsearch offers log and business analytics, full-text search, and security intelligence. The commonly used log analytics technology stack contains Elasticsearch, Logstash, and Kibana (ELK). ELK aggregates logs, performs analysis, and creates visualizations. AiCSD uses the Elasticsearch portion of the ELK stack for indexing the service container logs.

## Elasticsearch Authentication
Authentication is currently enabled for Elasticsearch. The Elastic Stack security features provide built-in user credentials to help bootstrap the stack setup. Elasticsearch users are initialized with the values of the passwords defined in the `.env` file.

## Resources

It isn't necessary to download Elasticsearch separately as it is included in the ELK stack log analytics feature of AiCSD. However, to find out more about Elasticsearch, see the resources below:

- [Elasticsearch](https://www.elastic.co/)
- [Elasticsearch Docs](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)


INTEL CONFIDENTIAL: See [License](../LICENSE.md).