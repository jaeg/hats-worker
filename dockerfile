FROM centos
COPY ./bin/hats-worker_unix /

ENTRYPOINT ["/hats-worker_unix"]
