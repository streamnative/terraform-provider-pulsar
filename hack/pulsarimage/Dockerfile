FROM apachepulsar/pulsar:4.0.3

ENV PULSAR_PREFIX_systemTopicEnabled=true
ENV PULSAR_PREFIX_topicLevelPoliciesEnabled=true
ENV PULSAR_PREFIX_enablePackagesManagement=true
ENV PULSAR_PREFIX_zookeeperServers="localhost:2181"
ENV PF_additionalEnabledConnectorUrlPatterns="*"
ENV PF_additionalEnabledFunctionsUrlPatterns="*"
RUN python3 /pulsar/bin/apply-config-from-env.py /pulsar/conf/standalone.conf
RUN python3 /pulsar/bin/gen-yml-from-env.py /pulsar/conf/functions_worker.yml

ENTRYPOINT ["/pulsar/bin/pulsar", "standalone"]
