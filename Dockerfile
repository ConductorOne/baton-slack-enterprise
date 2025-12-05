FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-slack-enterprise"]
COPY baton-slack-enterprise /
