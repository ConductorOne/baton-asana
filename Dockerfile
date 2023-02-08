FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-asana"]
COPY baton-asana /