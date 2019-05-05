FROM centos
COPY ./bin/wart_unix /
COPY ./bin/wart.config /

CMD ["/wart_unix", "--config=wart.config"]
