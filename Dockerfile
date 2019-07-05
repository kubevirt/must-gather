FROM quay.io/openshift/origin-must-gather:latest

# Save original gather script
RUN mv /usr/bin/gather /usr/bin/gather_original

# Use our gather script in place of the original one
COPY gather_cnv /usr/bin/gather
COPY gather_cnv_* /usr/bin/
COPY node-gather/node-gather-* /etc/

ENTRYPOINT /usr/bin/gather
