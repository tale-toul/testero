FROM registry.access.redhat.com/ubi8
ENV APPROOT=/go/bin
RUN mkdir -p $APPROOT && \
    chown -R 1001:0 $APPROOT &&\
    chmod -R g=u $APPROOT
COPY testero $APPROOT
USER 1001
WORKDIR $APPROOT
EXPOSE 8080
CMD ["./testero"]
